FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azure-bot-service-4.0 ./examples/azure/ai-bot-service/bot-service-4.0

ENTRYPOINT ["/azure-bot-service-4.0"]
