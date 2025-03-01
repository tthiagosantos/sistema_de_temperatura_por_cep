# Sistema de Temperatura por CEP

Este sistema é composto por **dois microserviços** instrumentados com OpenTelemetry e Zipkin para tracing distribuído. Ele permite consultar o clima (temperatura em Celsius, Fahrenheit e Kelvin) a partir de um CEP fornecido.

## Visão Geral

- **Serviço A (Input):**
    - **Endpoint:** `POST /cep`
    - **Função:** Recebe um CEP via JSON, valida se contém 8 dígitos e encaminha a requisição para o Serviço B.

- **Serviço B (Orquestração):**
    - **Endpoint:** `GET /weather?cep={cep}`
    - **Função:** Valida o CEP, consulta a API ViaCEP para identificar a cidade e a API WeatherAPI para obter a temperatura (em Celsius). Em seguida, converte a temperatura para Fahrenheit e Kelvin e retorna a resposta com o nome da cidade e os valores.

- **Observabilidade:**
    - Ambos os serviços estão instrumentados com OpenTelemetry e enviam os spans para um contêiner Zipkin.
    - O Zipkin está disponível em: `http://localhost:9411`

## Arquitetura

A solução é orquestrada via Docker Compose e possui a seguinte estrutura:

- sistema_de_temperatura_por_cep/
- ├── docker-compose.yaml
- ├── service-a/
- │   ├── Dockerfile
- │   ├── go.mod
- │   └── main.go
- └── service-b/
- ├── Dockerfile
- ├── go.mod
- └── main.go

Endpoints

Serviço A – Input
•	URL: http://localhost:8081/cep
•	Método: POST
•	Corpo da Requisição (JSON):
{ "cep": "29902555" }

•	Serviço B (teste direto):
Acesse:
http://localhost:8082/weather?cep=29902555

