FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /content-understanding-2025-11-01 ./examples/azure/ai-services/content-understanding-2025-11-01

ENTRYPOINT ["/content-understanding-2025-11-01"]
