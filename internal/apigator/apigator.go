// Package apigator defines all the functions and data structure for the
// interaction between the APIGatorDoraRouter and one or multiple APIGator
// instances
package apigator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	// MAX_ATTEMPTS represents the maximum number of attempts that the DoraRouter will try to reach a APIGator instance
	MAX_ATTEMPTS = 5
)

// APIGatorRouter defines the global configuration object for this Dora Router
// software. It also includes the list of APIGators to forward the request
type APIGatorRouter struct {
	Host            string `ini:"host"`
	Port            int    `ini:"port"`
	Path            string `ini:"path"`
	APIGatorTargets []*APIGatorTarget
}

// APIGatorTarget represents and APIGator server and authentication information for forwarding the incoming requests
type APIGatorTarget struct {
	Name         string `ini:"name"`
	Host         string `ini:"host"`
	Port         int    `ini:"port"`
	ClientID     string `ini:"client_id"`
	ClientSecret string `ini:"client_secret"`
	ApiKey       string `ini:"api_key"`
	Token        string
	Client       *http.Client
	Config       *APIGatorConfig
	Logger       *zap.Logger
}

// requestNewAccessToken uses the client_id and client_secret for obtainning a
// new Bearer Access Token from APIGator. If the response from APIGator is
// correct, it automatically saves the obtained token into the APIGatorTarget
// object. If it fails, and error is returned but the 'token' field is not updated
func (a *APIGatorTarget) requestNewAccessToken() error {
	a.Logger.Info("Requesting a new Access Token for APIGator", zap.String("apigator_target", a.Name))

	// Building Token HTTP Request Body
	data := url.Values{}
	data.Set("client_id", a.ClientID)
	data.Set("client_secret", a.ClientSecret)
	data.Set("grant_type", a.Config.GrantType)

	// Create the request body with the credentials
	req, err := http.NewRequest("POST", a.Host+a.Config.AuthPath, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	// Setting Token Request Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-API-Key", a.ApiKey)

	// Access Token HTTP Request
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}

	// Reading Token
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// If the Response code is 200OK, set the new token for the APIGatorTarget, if not, return err
	if resp.StatusCode == http.StatusOK {
		a.Logger.Info("Obtained new AccessToken for APIGator", zap.String("apigator_target", a.Name))
		var tokenResponse TokenResponse
		if err := json.Unmarshal(bodyBytes, &tokenResponse); err != nil {
			return err
		}
		a.Token = tokenResponse.AccessToken
	} else {
		return fmt.Errorf("Unexpected response status code (%d) when requesting a new AccessToken. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// UpdateRequestHeaders adds the needed HTTP headers to the incoming request
// for a correct interaction and authentication with an APIGator instance
func (a *APIGatorTarget) UpdateRequestHeaders(req *http.Request) error {
	if req == nil {
		return fmt.Errorf("Cannot Update HTTP headers on a NULL or empty request")
	}

	req.Header.Set("X-Resource-Token", "Bearer "+a.Token)
	req.Header.Set("X-API-Key", a.ApiKey)
	req.Header.Set("X-Data-Set-Type", "JSON")
	req.Header.Set("Content-Type", "application/json")

	return nil
}

// ForwardRequestToAPIGator takes an array of bytes as the body of a HTTP request and forwards it to its APIGator instance
// If the APIGator returns 401 (Unauthorized) it requests a new Access Token, creates a new request with the updated Headers and try again
// If the APIGator returns 400 (Bad Request) it creates a new request with the updated Headers and try again
// If the APIGator returns 200 (OK) it finishes and returns the response
func (a *APIGatorTarget) ForwardRequestToAPIGator(wg *sync.WaitGroup, body []byte, responseChan chan<- http.Response) error {
	for attempts := 0; attempts < MAX_ATTEMPTS; attempts++ {
		var resp *http.Response

		// Creating Request
		req, err := http.NewRequest("POST", a.Host+a.Config.DatasetPath, bytes.NewBuffer(body))
		if err != nil {
			a.Logger.Error("Failed to create request", zap.String("apigator_target", a.Name), zap.Error(err))
			return err
		}

		// Setting headers for APIGator
		if err := a.UpdateRequestHeaders(req); err != nil {
			return err
		}

		a.Logger.Debug("Performing HTTP Request on to APIGator",
			zap.String("apigator_target", a.Name),
			zap.String("url", req.URL.String()),
			zap.Int("try", attempts))

		// Forwarding HTTP request to APIGator
		resp, err = a.Client.Do(req)
		if err != nil {
			return err
		}

		// Checking the response Code
		if resp.StatusCode == http.StatusUnauthorized { // If there is no token yet, or the token has expired (401 Unauthorized)
			a.Logger.Warn("Token Expired for APIGator", zap.String("apigator_target", a.Name))
			if err := a.requestNewAccessToken(); err != nil {
				return err
			}
			continue
		} else if resp.StatusCode == http.StatusOK { // Response correct (200 OK)
			a.Logger.Debug("Response correct from APIGator")
			responseChan <- *resp
			return nil
		} else if resp.StatusCode > 400 && resp.StatusCode < 600 { // Every HTTP RC 4XX and 5XX
			respBodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("Request failed. Response Code: %d. HTTP response body: %s\n", resp.StatusCode, string(respBodyBytes))
		} else {
			a.Logger.Warn("Request is not correct. Trying again", zap.Int("status_code", resp.StatusCode))
			continue
		}
	}

	return fmt.Errorf("Maximmum tries reached. Request failed")
}
