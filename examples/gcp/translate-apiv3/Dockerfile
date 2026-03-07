FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /translate-apiv3 ./examples/gcp/translate-apiv3

ENTRYPOINT ["/translate-apiv3"]
