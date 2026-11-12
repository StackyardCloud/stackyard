FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /video-translation-2026-03-01 ./examples/azure/ai-services/video-translation-2026-03-01

ENTRYPOINT ["/video-translation-2026-03-01"]
