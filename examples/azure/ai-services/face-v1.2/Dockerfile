FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /face-v1.2 ./examples/azure/ai-services/face-v1.2

ENTRYPOINT ["/face-v1.2"]
