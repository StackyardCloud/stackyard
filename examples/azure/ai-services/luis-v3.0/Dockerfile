FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /luis-v3.0 ./examples/azure/ai-services/luis-v3.0

ENTRYPOINT ["/luis-v3.0"]
