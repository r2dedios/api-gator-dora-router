// Package apigator defines all the functions and data structure for the
// interaction between the APIGatorDoraRouter and one or multiple APIGator
// instances
package apigator

// APIGatorRouter defines the global configuration object for this Dora Router
// software. It also includes the list of APIGators to forward the request
type APIGatorRouter struct {
	Host            string `ini:"host"`
	Port            int    `ini:"port"`
	Path            string `ini:"path"`
	APIGatorTargets []*APIGatorTarget
	ScoreFuncName   string `ini:"score_function"`
	ScoreFunc       APIGatorResponseEvaluator
}
