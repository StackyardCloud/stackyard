FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /translator-v3.0 ./examples/azure/ai-services/translator-v3.0

ENTRYPOINT ["/translator-v3.0"]
