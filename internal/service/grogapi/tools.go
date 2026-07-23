package grogapi

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
)

var getWeatherTool = tool{
	Type: "function",
	Function: function{
		Name:        "get_weather",
		Description: "Get current weather for a location",
		Parameters: jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"location": {
					Type:        "string",
					Description: "City and state, e.g. San Francisco, CA",
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

type getWeatherReq struct {
	Location string `json:"location"`
	Unit     string `json:"unit"`
}

type getWeatherRes struct {
	Temperature int    `json:"location"`
	Condition   string `json:"condition"`
}

func getWeather(args string) string {
	var req getWeatherReq
	json.Unmarshal([]byte(args), &req)

	res := getWeatherRes{
		Temperature: 77,
		Condition:   "cloudy",
	}

	if req.Unit == "celcius" {
		res.Temperature = 25
	}

	switch req.Location {
	case "San Fransisco, CA":
		res.Condition = "sunny"
	case "Medan, North Sumatra":
		res.Condition = "rainy"
	}

	v, _ := json.Marshal(res)
	return string(v)
}
