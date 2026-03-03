FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /prometheus ./examples/aws/prometheus/prometheus
ENTRYPOINT ["/prometheus"]
