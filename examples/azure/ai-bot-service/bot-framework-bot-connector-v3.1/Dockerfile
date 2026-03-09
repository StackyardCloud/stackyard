FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /bot-framework-bot-connector-v3-1 ./examples/azure/ai-bot-service/bot-framework-bot-connector-v3.1

ENTRYPOINT ["/bot-framework-bot-connector-v3-1"]
