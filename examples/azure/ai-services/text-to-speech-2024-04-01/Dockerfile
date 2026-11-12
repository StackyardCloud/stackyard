FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /text-to-speech-2024-04-01 ./examples/azure/ai-services/text-to-speech-2024-04-01

ENTRYPOINT ["/text-to-speech-2024-04-01"]
