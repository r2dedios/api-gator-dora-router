package apigator

import (
	"encoding/json"
	"go.uber.org/zap"
	"strings"
)

const (
	bestResponseScore  = 1.0
	worstResponseScore = 0.0
)

// APIGatorResponseEvaluator defines the structure of the functions used the
// different strategies for evaluating which is the "best" response from
// APIGator and return it to the original requester.
// Each function for evaluating response must return a score from 0 to 1 where
// higher numbers means "better" response
type APIGatorResponseEvaluator func(data map[string]interface{}, logger *zap.Logger, restrictedText string) float64

// BasicEvaluator considers a response as valid if the 'dataSet' key exists or not
func BasicEvaluator(data map[string]interface{}, logger *zap.Logger, restrictedText string) float64 {
	if _, exists := data["dataSet"]; exists {
		return bestResponseScore
	}
	logger.Error("response does not contain the 'dataSet' key")
	return worstResponseScore
}

/*
CountStringOccurrences scans a given JSON document (represented as a map) and counts the occurrences
of a specified string within the values of the JSON. It also counts the total number of parameters scanned.

Parameters:
- jsonMap (map[string]interface{}): The JSON document already unmarshaled into a map.
- targetString (string): The string to search for within the values of the JSON document.

Returns:
- int: The number of times the target string appears in the JSON values.
- int: The total number of parameters scanned in the JSON document.

This function traverses the JSON document recursively to ensure all nested values are checked.
*/
func countObjectKeys(data map[string]interface{}, pattern string) (int, int) {
	var count int = 0
	var total int = 0

	for _, value := range data {
		total++
		// Switching based on value type
		switch v := value.(type) {
		// if string, check the value
		case string:
			if strings.Contains(value.(string), pattern) {
				count++
			}
		// if nested JSON, recursive calling to this function
		case map[string]interface{}:
			a, b := countObjectKeys(v, pattern)
			count += a
			total += b
		// if embedded JSON parse the value as a new JSON doc, and continue recursive calling
		case []interface{}:
			for _, item := range v {
				if subMap, ok := item.(map[string]interface{}); ok {
					a, b := countObjectKeys(subMap, pattern)
					count += a
					total += b
				}
			}
		}
	}

	return count, total
}

// PercentEvaluator evaluates the percent of the fields that are de/crypted in a APIGator Response
func PercentEvaluator(data map[string]interface{}, logger *zap.Logger, restrictedText string) float64 {
	var dataSet map[string]interface{}

	dataBytes := []byte(data["dataSet"].(string))
	err := json.Unmarshal(dataBytes, &dataSet)
	if err != nil {
		logger.Error("Failed to Unmarshal response body", zap.Error(err))
		return -1.0
	}

	count, total := countObjectKeys(dataSet, restrictedText)
	score := float64(count) / float64(total)

	// The returned value is 1-score because less crypted values increases the final score of the response
	return 1 - score
}
