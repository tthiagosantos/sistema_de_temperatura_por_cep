FROM golang:1.20-bullseye as build
WORKDIR /app
COPY . .
RUN apt-get update && apt-get install -y ca-certificates
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service-b main.go

FROM scratch
COPY --from=build /app/service-b .
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8082
ENTRYPOINT ["./service-b"]