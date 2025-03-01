package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

var tracer trace.Tracer

func main() {
	initTracerProvider()

	http.HandleFunc("/cep", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "ServiceA /cep Handler")
		defer span.End()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"message": "use POST"})
			return
		}

		var body struct {
			CEP string `json:"cep"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
			return
		}

		// Valida que o CEP tenha exatamente 8 dígitos
		if matched, _ := regexp.MatchString(`^[0-9]{8}$`, body.CEP); !matched {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"message": "invalid zipcode"})
			return
		}

		// Chama o Serviço B para obter o clima
		resp, err := callServiceB(ctx, body.CEP)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Propaga status e body do serviço B
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	log.Println("Service A iniciando na porta 8081...")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

func callServiceB(ctx context.Context, cep string) (*http.Response, error) {
	ctx, span := tracer.Start(ctx, "ServiceA callServiceB")
	defer span.End()

	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://service-b:8082"
	}

	url := fmt.Sprintf("%s/weather?cep=%s", serviceBURL, cep)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
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
			semconv.ServiceName("service-a"),
		)),
	)
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("service-a-tracer")
}
