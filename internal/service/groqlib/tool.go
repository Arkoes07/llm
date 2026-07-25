package groqlib

import "github.com/conneroisu/groq-go/pkg/tools"

var getWeatherTool = tools.Tool{
	Type: "function",
	Function: tools.FunctionDefinition{
		Name:        "get_weather",
		Description: "Get current weather for a location, by default return celcius",
		Parameters: tools.FunctionParameters{
			Type: "object",
			Properties: map[string]tools.PropertyDefinition{
				"location": {
					Type:        "string",
					Description: "City and state/province, e.g. Medan, North Sumatra",
				},
				"unit": {
					Type:        "string",
					Description: "Known enum: 'celsius' and 'fahrenheit'",
				},
			},
			Required: []string{"location"},
		},
	},
}

var queryLogsTool = tools.Tool{
	Type: "function",
	Function: tools.FunctionDefinition{
		Name:        "query_logs",
		Description: "Retrieves raw application log lines for a single service over a time window. Returns timestamped lines including log level (INFO/WARN/ERROR) and message text, in chronological order. Use this first to find error messages and warnings that describe the symptom. Does not return aggregated numbers — use get_metrics for those.",
		Parameters: tools.FunctionParameters{
			Type: "object",
			Properties: map[string]tools.PropertyDefinition{
				"service": {
					Type:        "string",
					Description: "Name of the service, e.g. 'checkout', 'payment', 'inventory'.",
				},
				"time_range": {
					Type:        "string",
					Description: "Time window in 24h HH:MM-HH:MM format, e.g. '14:00-14:05'. Keep the window narrow, within a few minutes of the reported symptom.",
				},
			},
			Required: []string{"service", "time_range"},
		},
	},
}

var getMetricsTool = tools.Tool{
	Type: "function",
	Function: tools.FunctionDefinition{
		Name:        "get_metrics",
		Description: "Retrieves current numeric metric values for a single service. Returns the metric value with its unit and whether it is within normal range. Use this to confirm or rule out a hypothesis formed from the logs — for example, to check whether a resource is saturated or whether traffic is abnormal. Returns numbers only, no log text.",
		Parameters: tools.FunctionParameters{
			Type: "object",
			Properties: map[string]tools.PropertyDefinition{
				"service": {
					Type:        "string",
					Description: "Name of the service, e.g. 'checkout', 'payment', 'inventory'.",
				},
				"metric": {
					Type:        "string",
					Description: "Which metric to fetch. Known enum: 'db_pool', 'latency', 'error_rate', 'request_rate'",
				},
			},
			Required: []string{"service", "metric"},
		},
	},
}

var searchRunbookTool = tools.Tool{
	Type: "function",
	Function: tools.FunctionDefinition{
		Name:        "search_runbook",
		Description: "Searches the internal operations runbooks by keyword and returns the single best-matching runbook entry, containing its symptom description, known cause, and remediation steps. Use this after identifying a specific symptom to find the documented fix. Search using the symptom or error text, not the service name.",
		Parameters: tools.FunctionParameters{
			Type: "object",
			Properties: map[string]tools.PropertyDefinition{
				"query": {
					Type:        "string",
					Description: "Symptom keywords or error text to search for, e.g. 'pool exhausted', 'connection timeout'. Not a service name.",
				},
			},
			Required: []string{"query"},
		},
	},
}
