FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /ai-services-language-analyze-text-authoring-2023-04-01 ./examples/azure/ai-services/language-analyze-text-authoring-2023-04-01

ENTRYPOINT ["/ai-services-language-analyze-text-authoring-2023-04-01"]
