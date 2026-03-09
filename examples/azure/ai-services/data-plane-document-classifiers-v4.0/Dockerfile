FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /ai-services-data-plane-document-classifiers-v4.0 ./examples/azure/ai-services/data-plane-document-classifiers-v4.0

ENTRYPOINT ["/ai-services-data-plane-document-classifiers-v4.0"]
