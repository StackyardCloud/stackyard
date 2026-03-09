FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /bot-framework-direct-line-v3-0 ./examples/azure/ai-bot-service/bot-framework-direct-line-v3.0

ENTRYPOINT ["/bot-framework-direct-line-v3-0"]
