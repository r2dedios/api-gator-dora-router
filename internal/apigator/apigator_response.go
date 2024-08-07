package apigator

import (
	"bytes"
	"encoding/json"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strings"
)

// APIGatorResponse represents a response from an APIGator instance. It
// contains the HTTP response and the name of the APIGator instances which
// returned the response
type APIGatorResponse struct {
	Response http.Response
	Name     string
}

// EvaluateResponse evaulates a HTTP response is valid or not using the
// funciton referenced by args and returns a boolean value with the result
func (r *APIGatorResponse) EvaluateResponse(fp APIGatorResponseEvaluator, restrictedText string, original string, logger *zap.Logger) float64 {
	var responseData map[string]interface{}

	// Getting Response body as []bytes
	respBodyBytes, err := ioutil.ReadAll(r.Response.Body)
	if err != nil {
		return -1.0
	}
	// Restore Request Body because it was supposed to be read just once
	r.Response.Body = ioutil.NopCloser(bytes.NewBuffer(respBodyBytes))

	// Unpackaging Response into JSON format
	err = json.Unmarshal(respBodyBytes, &responseData)
	if err != nil {
		logger.Error("Failed to Unmarshal response body", zap.Error(err))
		return -1.0
	}

	// if the response is the same as the received request, it's discard
	dataSet := responseData["dataSet"].(string)
	if !isResponseModified(dataSet, original) {
		logger.Debug("Detected Response without any change. Discarding...",
			zap.String("apigator", r.Name),
		)
		return -1.0
	}

	// As the response from APIGator is always 200(OK) independently if it was
	// able to decrypt the payload or not, the evaluation is performed based on a
	// specific method configured on the INI file
	if r.Response.StatusCode == http.StatusOK {
		return fp(responseData, logger, restrictedText)
	}
	return -1.0
}

// isResponseModified compares the responseBody and original strings and returns a
// boolean value indicating if both parameters has the same value or not, which
// indicates that the response was not modified by APIGator, and shouldn't be
// considered as the "best" response
func isResponseModified(responseBody string, original string) bool {
	str := strings.Replace(responseBody, "\n", "", -1)
	str = strings.Replace(str, " ", "", -1)
	str = strings.Replace(str, "\"", "'", -1)

	if str == original {
		return false
	}
	return true
}
