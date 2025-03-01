FROM golang:1.20-bullseye as build
WORKDIR /app
COPY . .
RUN apt-get update && apt-get install -y ca-certificates
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service-a main.go

FROM scratch
COPY --from=build /app/service-a .
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8081
ENTRYPOINT ["./service-a"]