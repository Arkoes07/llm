package mocktool

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

func Run(name string, args string) (string, error) {
	switch name {
	case "get_weather":
		var req getWeatherReq
		if err := json.Unmarshal([]byte(args), &req); err != nil {
			return "", fmt.Errorf("bad arguments for get_weather: %v", err)
		}

		res, _ := json.Marshal(getWeather(req))
		return string(res), nil
	case "query_logs":
		var req queryLogsReq
		if err := json.Unmarshal([]byte(args), &req); err != nil {
			return "", fmt.Errorf("bad arguments for query_logs: %v", err)
		}

		return queryLogs(req)
	case "get_metrics":
		var req getMetricsReq
		if err := json.Unmarshal([]byte(args), &req); err != nil {
			return "", fmt.Errorf("bad arguments for get_metrics: %v", err)
		}

		return getMetrics(req)

	case "search_runbook":
		var req searchRunbookReq
		if err := json.Unmarshal([]byte(args), &req); err != nil {
			return "", fmt.Errorf("bad arguments for search_runbook: %v", err)
		}

		return searchRunbook(req)
	}

	return "", fmt.Errorf("unknown tool %q", name)
}

type getWeatherReq struct {
	Location string `json:"location"`
	Unit     string `json:"unit"`
}

type getWeatherRes struct {
	Temperature int    `json:"location"`
	Condition   string `json:"condition"`
}

func getWeather(args getWeatherReq) getWeatherRes {
	temp := rand.Intn(15) + 20
	condition := "sunny"
	if temp < 30 {
		condition = "cloudy"
	} else if temp < 25 {
		condition = "rainy"
	}

	if args.Unit == "fahrenheit" {
		temp = (temp * 9 / 5) + 32
	}

	return getWeatherRes{
		Temperature: temp,
		Condition:   condition,
	}
}
