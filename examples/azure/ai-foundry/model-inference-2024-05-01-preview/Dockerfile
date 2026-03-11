FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /model-inference-2024-05-01-preview ./examples/azure/ai-foundry/model-inference-2024-05-01-preview

ENTRYPOINT ["/model-inference-2024-05-01-preview"]
