FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /speech-to-text-v3.2-preview.2 ./examples/azure/ai-services/speech-to-text-v3.2-preview.2

ENTRYPOINT ["/speech-to-text-v3.2-preview.2"]
