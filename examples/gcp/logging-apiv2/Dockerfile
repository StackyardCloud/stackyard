FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /logging-apiv2 ./examples/gcp/logging-apiv2

ENTRYPOINT ["/logging-apiv2"]
