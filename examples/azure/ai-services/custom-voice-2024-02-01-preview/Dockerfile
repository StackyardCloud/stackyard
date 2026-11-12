FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /custom-voice-2024-02-01-preview ./examples/azure/ai-services/custom-voice-2024-02-01-preview

ENTRYPOINT ["/custom-voice-2024-02-01-preview"]
