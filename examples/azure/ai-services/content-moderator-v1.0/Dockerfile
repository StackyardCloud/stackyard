FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /content-moderator-v1.0 ./examples/azure/ai-services/content-moderator-v1.0

ENTRYPOINT ["/content-moderator-v1.0"]
