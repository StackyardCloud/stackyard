FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /application-gateway-containers-2025-01-01 ./examples/azure/application-gateway-containers-2025-01-01

ENTRYPOINT ["/application-gateway-containers-2025-01-01"]
