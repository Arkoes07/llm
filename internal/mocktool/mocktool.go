package mocktool

import (
	"encoding/json"
	"math/rand"
)

func Run(name string, args string) string {
	switch name {
	case "get_weather":
		return getWeather(args)
	}

	return ""
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
	temp := rand.Intn(15) + 20
	condition := "sunny"
	if temp < 30 {
		condition = "cloudy"
	} else if temp < 25 {
		condition = "rainy"
	}

	var req getWeatherReq
	json.Unmarshal([]byte(args), &req)
	if req.Unit == "fahrenheit" {
		temp = (temp * 9 / 5) + 32
	}

	res, _ := json.Marshal(getWeatherRes{
		Temperature: temp,
		Condition:   condition,
	})

	return string(res)
}
