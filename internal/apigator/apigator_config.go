package apigator

import (
	"time"
)

const (
	defaultDatasetPath = "/apigator/protect/v1/dataset"
	defaultAuthPath    = "/apigator/identity/v1/token"
	defaultGrantType   = "client_credentials"
	// ammount of seconds for throwing a Timeout error
	defaultAPIGatorTimeout = 60 * time.Second
)

// APIGatorConfig represents basic APIGator configuration in common for every APIGatorTarget
type APIGatorConfig struct {
	DatasetPath string        `ini:"dataset_path"`
	AuthPath    string        `ini:"auth_path"`
	GrantType   string        `ini:"grant_type"`
	Timeout     time.Duration `ini:"timeout"`
}
