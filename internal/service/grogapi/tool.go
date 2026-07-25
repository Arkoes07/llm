package grogapi

var getWeatherTool = tool{
	Type: "function",
	Function: toolFunc{
		Name:        "get_weather",
		Description: "Get current weather for a location, by default return celcius",
		Parameters: funcParam{
			Type: "object",
			Properties: map[string]property{
				"location": {
					Type:        "string",
					Description: "City and state/province, e.g. Medan, North Sumatra",
				},
				"unit": {
					Type: "string",
					Enum: []any{"celsius", "fahrenheit"},
				},
			},
			Required: []string{"location"},
		},
	},
}

var queryLogsTool = tool{
	Type: "function",
	Function: toolFunc{
		Name:        "query_logs",
		Description: "Retrieves raw application log lines for a single service over a time window. Returns timestamped lines including log level (INFO/WARN/ERROR) and message text, in chronological order. Use this first to find error messages and warnings that describe the symptom. Does not return aggregated numbers — use get_metrics for those.",
		Parameters: funcParam{
			Type: "object",
			Properties: map[string]property{
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

var getMetricsTool = tool{
	Type: "function",
	Function: toolFunc{
		Name:        "get_metrics",
		Description: "Retrieves current numeric metric values for a single service. Returns the metric value with its unit and whether it is within normal range. Use this to confirm or rule out a hypothesis formed from the logs — for example, to check whether a resource is saturated or whether traffic is abnormal. Returns numbers only, no log text.",
		Parameters: funcParam{
			Type: "object",
			Properties: map[string]property{
				"service": {
					Type:        "string",
					Description: "Name of the service, e.g. 'checkout', 'payment', 'inventory'.",
				},
				"metric": {
					Type:        "string",
					Description: "Which metric to fetch.",
					Enum:        []any{"db_pool", "latency", "error_rate", "request_rate"},
				},
			},
			Required: []string{"service", "metric"},
		},
	},
}

var searchRunbookTool = tool{
	Type: "function",
	Function: toolFunc{
		Name:        "search_runbook",
		Description: "Searches the internal operations runbooks by keyword and returns the single best-matching runbook entry, containing its symptom description, known cause, and remediation steps. Use this after identifying a specific symptom to find the documented fix. Search using the symptom or error text, not the service name.",
		Parameters: funcParam{
			Type: "object",
			Properties: map[string]property{
				"query": {
					Type:        "string",
					Description: "Symptom keywords or error text to search for, e.g. 'pool exhausted', 'connection timeout'. Not a service name.",
				},
			},
			Required: []string{"query"},
		},
	},
}
