// Package main contains the main and auxiliar functions for the APIGatorDoraRouter.
package main

import (
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
	responseChan := make(chan ag.APIGatorResponse, len(router.APIGatorTargets))
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
	var restrictedText string
	if jsonData != nil {
		restrictedText = jsonData["restrictedText"].(string)
	} else {
		logger.Error("Incoming data JSON is empty!")
		return
	}

	logger.Debug("Processing responses", zap.String("restricted_text", restrictedText))
	resp := processResponses(responseChan, restrictedText, jsonData["dataSet"].(string))
	if resp == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "No response"})
		return
	}
	respBodyBytes, err := ioutil.ReadAll(resp.Response.Body)
	if err != nil {
		return
	}

	// Responding best response
	logger.Info("Responding back to requester",
		zap.String("apigator_target", resp.Name),
	)
	c.Writer.Write(respBodyBytes)
}

// processResponses reads every response obtained from the list of
// APIGatorsTargets, and selects which is the best one based on the evaluation
// function defined on the router's configuration
func processResponses(responseChan <-chan ag.APIGatorResponse, restrictedText string, original string) *ag.APIGatorResponse {
	// TODO: test with pointer
	var bestResponse ag.APIGatorResponse
	var bestScore float64 = 0.0
	var name string

	// Reading every response and evaulatin its score
	for r := range responseChan {
		defer r.Response.Body.Close()
		score := r.EvaluateResponse(router.ScoreFunc, restrictedText, original, logger)
		logger.Debug("Evaluating Response", zap.Float64("score", score))
		if score > bestScore {
			logger.Debug("New Best Response", zap.Float64("score", score))
			bestScore = score
			bestResponse = r
			name = r.Name
		}
	}

	logger.Debug("Selected Response from APIGator",
		zap.String("score_method", router.ScoreFuncName),
		zap.String("apigator", name),
		zap.Float64("score", bestScore),
	)
	return &bestResponse
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
