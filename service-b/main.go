package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	url2 "net/url"
	"os"
	"regexp"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrCEPNotFound = errors.New("cep not found")
	weatherAPIKey  string
	tracer         trace.Tracer
)

func main() {
	initTracerProvider()
	weatherAPIKey = os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		log.Fatal("WEATHER_API_KEY não definida")
	}

	http.HandleFunc("/weather", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "ServiceB /weather Handler")
		defer span.End()

		cep := r.URL.Query().Get("cep")
		// Valida que o CEP tenha 8 dígitos
		if matched, _ := regexp.MatchString(`^[0-9]{8}$`, cep); !matched {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"message": "invalid zipcode"})
			return
		}

		city, err := fetchCityFromCEP(ctx, cep)
		if err != nil {
			if errors.Is(err, ErrCEPNotFound) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"message": "can not find zipcode"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}

		tempC, err := fetchTemperatureCelsius(ctx, city)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}

		tempF := celsiusToFahrenheit(tempC)
		tempK := celsiusToKelvin(tempC)

		resp := map[string]interface{}{
			"city":   city,
			"temp_C": tempC,
			"temp_F": tempF,
			"temp_K": tempK,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	log.Println("Service B iniciando na porta 8082...")
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatal(err)
	}
}

func fetchCityFromCEP(ctx context.Context, cep string) (string, error) {
	ctx, span := tracer.Start(ctx, "fetchCityFromCEP")
	defer span.End()

	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ViaCEP status: %d", resp.StatusCode)
	}

	var data struct {
		Localidade string `json:"localidade"`
		Erro       bool   `json:"erro"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Erro {
		return "", ErrCEPNotFound
	}

	return data.Localidade, nil
}

func fetchTemperatureCelsius(ctx context.Context, city string) (float64, error) {
	ctx, span := tracer.Start(ctx, "fetchTemperatureCelsius")
	defer span.End()

	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s", weatherAPIKey, url2.QueryEscape(city))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("WeatherAPI status: %d", resp.StatusCode)
	}

	var data struct {
		Current struct {
			TempC float64 `json:"temp_c"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return data.Current.TempC, nil
}

func celsiusToFahrenheit(c float64) float64 {
	return c*1.8 + 32
}

func celsiusToKelvin(c float64) float64 {
	return c + 273
}

func initTracerProvider() {
	zipkinURL := os.Getenv("ZIPKIN_ENDPOINT")
	if zipkinURL == "" {
		zipkinURL = "http://zipkin:9411/api/v2/spans"
	}
	exporter, err := zipkin.New(zipkinURL)
	if err != nil {
		log.Fatalf("Erro ao criar exporter Zipkin: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("service-b"),
		)),
	)
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("service-b-tracer")
}
