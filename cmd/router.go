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
	router  *ag.APIGatorRouter
	logger  *zap.Logger
	gRouter *gin.Engine
)

const (
	healthcheckPath = "/health"
)

func init() {
	logger = gLogger.NewLogger()
	gin.SetMode(gin.ReleaseMode)
	gRouter = gin.New()
	gRouter.Use(ginzap.Ginzap(logger, time.RFC3339, true))
}

func healthcheck(c *gin.Context) {
	logger.Info("Healthcheck probe requested")
	c.JSON(http.StatusOK, gin.H{"health_status": "ok"})
}

func isRequestCorrect(resp *http.Response) bool {
	var responseData map[string]interface{}

	// Getting Response body as []bytes
	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// Restore Request Body
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBodyBytes))

	err = json.Unmarshal(respBodyBytes, &responseData)
	if err != nil {
		logger.Error("Failed to Unmarshal response body", zap.Error(err))
		return false
	}

	// If the key exists, return the response
	if _, exists := responseData["dataSet"]; exists && resp.StatusCode == http.StatusOK {
		return true
	}

	logger.Error("response does not contain the 'dataSet' key")
	return false
}

func processResponses(responseChan <-chan http.Response) *http.Response {
	for resp := range responseChan {
		if isRequestCorrect(&resp) {
			return &resp
		}
	}
	return nil
}

func forwardRequest(c *gin.Context) {
	logger.Debug("Received Request", zap.String("origin", c.RemoteIP()))

	// Obtainning JSON body from request
	var jsonData map[string]interface{}
	if err := c.BindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

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

		// Simultaneous forwarding
		wg.Add(1)
		go func(id int, apiGator *ag.APIGatorTarget) {
			logger.Debug("Launching Forwarding thread", zap.Int("id", id))
			if err := apiGator.ForwardRequestToAPIGator(&wg, jsonBytes, responseChan); err != nil {
				logger.Error("Failed to send request", zap.String("apigator_target", apiGator.Name), zap.Error(err))
			}
			wg.Done()
		}(i, apiGator)

	}
	// Wait for all requests to complete
	wg.Wait()
	defer close(responseChan)

	resp := processResponses(responseChan)
	if resp == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "No response"})
		return
	}
	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
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
	gRouter.GET(healthcheckPath, healthcheck)
	listenAddress := router.Host + ":" + fmt.Sprintf("%d", router.Port)

	logger.Info("Listening for requests")
	if err := gRouter.Run(listenAddress); err != nil {
		logger.Fatal("Failed to run APIGator Dora Router", zap.Error(err))
	}
	os.Exit(0)
}
