// Package main contains the main and auxiliar functions for the APIGatorDoraRouter.
package main

import (
	"bytes"
	"encoding/json"
	ag "exate-dora-router/internal/apigator"
	cfg "exate-dora-router/internal/config"
	gLogger "exate-dora-router/internal/logger"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
)

var (
	// Router config struct for obtaining the configuration parameters of this
	// router and the list of targets for broadcasting the incoming requests
	router *ag.APIGatorRouter

	// logger for the APIGatorDoraRouter. It's used across all this binary
	logger *zap.Logger

	// gRouter object for defining the GinFramework router instance for the HTTP server
	gRouter *gin.Engine
)

const (
	// URL path for the Healthcheck handler. This was included for the K8s probes.
	healthcheckPath = "/healthz"
)

// Init function for pre-configuring the global vars for the router
func init() {
	// Creating a new instance for the logger
	logger = gLogger.NewLogger()

	// Configures and creates a new Instance of the Gin Router for the HTTP server
	gin.SetMode(gin.ReleaseMode)
	gRouter = gin.New()

	// Attaching the logger to the GIN server. This will forward Gin's logs to
	// the same logger maintainning the structure and the output channels for
	// logs
	gRouter.Use(ginzap.Ginzap(logger, time.RFC3339, true))
}

// healthcheckHandler manages the incoming connections on the path "/healthz"
// for evaulating K8s Startup/Readiness/Liveness probes
func healthcheckHandler(c *gin.Context) {
	logger.Info("Healthcheck probe requested")
	c.JSON(http.StatusOK, gin.H{"health_status": "ok"})
}

// evaluateResponse evaulates a HTTP response is valid or not using the
// funciton referenced by args and returns a boolean value with the result
func evaluateResponse(resp *http.Response, fp ag.APIGatorResponseEvaluator, restrictedText string) float64 {
	var responseData map[string]interface{}

	// Getting Response body as []bytes
	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1.0
	}
	// Restore Request Body because it was supposed to be read just once
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBodyBytes))

	// Unpackaging Response into JSON format
	err = json.Unmarshal(respBodyBytes, &responseData)
	if err != nil {
		logger.Error("Failed to Unmarshal response body", zap.Error(err))
		return -1.0
	}

	// As the response from APIGator is always 200(OK) independently if it was
	// able to decrypt the payload or not, the evaluation is performed based on a
	// specific method configured on the INI file
	if resp.StatusCode == http.StatusOK {
		return fp(responseData, logger, restrictedText)
	}
	return -1.0
}

// processResponses reads every response obtained from the list of
// APIGatorsTargets, and selects which is the best one based on the evaluation
// function defined on the router's configuration
func processResponses(responseChan <-chan http.Response, restrictedText string) *http.Response {
	var bestResponse http.Response
	var bestScore float64 = 0.0

	// Reading every response and evaulatin its score
	for resp := range responseChan {
		defer resp.Body.Close()
		score := evaluateResponse(&resp, router.ScoreFunc, restrictedText)
		logger.Debug("Evaluating Response", zap.Float64("score", score))
		if score > bestScore {
			logger.Debug("New Best Response", zap.Float64("score", score))
			bestScore = score
			bestResponse = resp
		}
	}
	logger.Debug("Selected Response from APIGator", zap.String("score_method", router.ScoreFuncName), zap.Float64("score", bestScore))
	return &bestResponse
}

// forwardRequest is the main HTTP handler function for the APIGatorDoraRouter.
// It takes the incoming requests with the data to process, and forwards it to
// every APIGator target defined on the config.ini file.
// Depending on the scoring method chosen, the router will select a response
// from among all those received by the different targets, to return it to the
// originating requester.
func forwardRequest(c *gin.Context) {
	// Logging the origin IP of the requester
	logger.Debug("Received Request", zap.String("origin", c.RemoteIP()))

	// Obtainning JSON body from request
	var jsonData map[string]interface{}
	if err := c.BindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Creating channel and WaitGroup for forwarding the request to the APIGatorTarget list in parallel
	responseChan := make(chan http.Response, len(router.APIGatorTargets))
	var wg sync.WaitGroup

	// Forwarding to the list of APIGator instances simultaneously
	for i, _ := range router.APIGatorTargets {
		apiGator := router.APIGatorTargets[i]
		logger.Debug("Forwarding request to APIGator instance", zap.String("apigator_target", apiGator.Name))
		// Decoding JSON request body
		jsonBytes, err := json.Marshal(jsonData)
		if err != nil {
			logger.Error("Failed to marshal JSON", zap.String("apigator_target", apiGator.Name), zap.Error(err))
			return
		}

		// Simultaneous forwarding on parallel. Creating one thread per APIGator target
		wg.Add(1)
		//TODO: remove sleep. It's just for testing
		time.Sleep(1 * time.Second)
		go func(id int, apiGator *ag.APIGatorTarget) {
			logger.Debug("Launching Forwarding thread", zap.Int("id", id))
			if err := apiGator.ForwardRequestToAPIGator(&wg, jsonBytes, responseChan); err != nil {
				logger.Error("Failed to send request", zap.String("apigator_target", apiGator.Name), zap.String("request_body", string(jsonBytes)), zap.Error(err))
			}
			wg.Done()
		}(i, apiGator)

	}
	// Wait for all requests to complete and closes the channel once every thread has answered
	wg.Wait()
	close(responseChan)

	// Get 'restrictedText' field for each request. It will be used later for evaluating the response score
	restrictedText := jsonData["restrictedText"].(string)

	logger.Debug("Processing responses", zap.String("restricted_text", restrictedText))
	resp := processResponses(responseChan, restrictedText)
	if resp == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "No response"})
		return
	}
	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Responding best response
	c.Writer.Write(respBodyBytes)
}

func main() {
	// Ignore Logger sync error
	defer func() { _ = logger.Sync() }()

	logger.Info("Starting APIGator Dora Router for Exate")
	logger.Debug("Debug Mode active!")

	// Define the command-line flags
	configFilePath := flag.String("config", "config.ini", "Path to the INI configuration file")
	flag.Parse()

	// Load the configuration
	var err error
	router, err = cfg.LoadConfig(*configFilePath, logger)
	logger.Debug("Loading INI config file", zap.String("config_file", *configFilePath))
	if err != nil {
		logger.Fatal("Can't read INI config file", zap.Error(err))
	}

	gRouter.POST(router.Path, forwardRequest)
	gRouter.GET(healthcheckPath, healthcheckHandler)
	listenAddress := router.Host + ":" + fmt.Sprintf("%d", router.Port)

	logger.Info("Listening for requests")
	if err := gRouter.Run(listenAddress); err != nil {
		logger.Fatal("Failed to run APIGator Dora Router", zap.Error(err))
	}
	os.Exit(0)
}
