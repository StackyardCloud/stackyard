FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /health-insights-2024-10-01 ./examples/azure/ai-services/health-insights-2024-10-01

ENTRYPOINT ["/health-insights-2024-10-01"]
