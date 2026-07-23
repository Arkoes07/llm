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
