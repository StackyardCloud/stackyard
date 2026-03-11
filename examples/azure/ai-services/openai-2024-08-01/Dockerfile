FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /openai-2024-08-01 ./examples/azure/ai-services/openai-2024-08-01

ENTRYPOINT ["/openai-2024-08-01"]
