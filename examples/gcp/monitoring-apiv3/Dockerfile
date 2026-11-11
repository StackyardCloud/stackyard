FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /monitoring-apiv3 ./examples/gcp/monitoring-apiv3

ENTRYPOINT ["/monitoring-apiv3"]
