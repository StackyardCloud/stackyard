FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /openai-2024-10-21 ./examples/azure/ai-services/openai-2024-10-21

ENTRYPOINT ["/openai-2024-10-21"]
