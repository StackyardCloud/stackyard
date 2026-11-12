FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /content-safety-2024-09-01 ./examples/azure/ai-services/content-safety-2024-09-01

ENTRYPOINT ["/content-safety-2024-09-01"]
