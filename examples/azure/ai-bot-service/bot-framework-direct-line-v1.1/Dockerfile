FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /bot-framework-direct-line-v1-1 ./examples/azure/ai-bot-service/bot-framework-direct-line-v1.1

ENTRYPOINT ["/bot-framework-direct-line-v1-1"]
