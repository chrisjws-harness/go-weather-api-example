package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	harness "github.com/harness/ff-golang-server-sdk/client"
	"github.com/harness/ff-golang-server-sdk/evaluation"
)

var (
	flagName string = "default_imperial"
	apiKeyFF string = os.Getenv("FF_API_KEY")
)

type WeatherData struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Name string `json:"name"`
}

func getWeatherData(city string, apiKey string, geography ...string) (*WeatherData, error) {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=%s", city, apiKey, geography[0])
	fmt.Println(geography)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var weatherData WeatherData
	err = json.Unmarshal(body, &weatherData)
	if err != nil {
		return nil, err
	}

	return &weatherData, nil
}

func weatherHandler(w http.ResponseWriter, r *http.Request, apiKey string, ff_client harness.CfClient) {
	city := r.FormValue("city")
	if city == "" {
		http.Error(w, "City not specified", http.StatusBadRequest)
		return
	}

	loc := r.Header.Get("user-location")
	username := r.Header.Get("whoami")

	fmt.Println(loc, username)
	target := evaluation.Target{
		Identifier: "user1",
		Name:       "user1",
		Attributes: &map[string]interface{}{"location": "usa"},
	}

	resultBool, err := ff_client.BoolVariation(flagName, &target, false)
	if err != nil {
		log.Fatal("failed to get evaluation: ", err)
	}
	log.Printf("Flag variation %v\n", resultBool)
	geo := r.Header.Get("geography")
	if geo != "imperial" && geo != "metric" {
		if resultBool == true {
			geo = "imperial"
		} else {
			geo = "metric"
		}
	}
	fmt.Println(geo)

	weatherData, err := getWeatherData(city, apiKey, geo)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching weather data: %s", err), http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(weatherData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshaling JSON response: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func main() {
	apiKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OpenWeatherMap API key not provided.")
		return
	}

	client, err := harness.NewCfClient(apiKeyFF)
	if err != nil {
		log.Fatalf("could not connect to CF servers %s\n", err)
	}
	defer func() { client.Close() }()

	r := mux.NewRouter()
	r.HandleFunc("/weather", func(w http.ResponseWriter, r *http.Request) {
		weatherHandler(w, r, apiKey, *client)
	}).Methods("GET")

	http.Handle("/", r)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
