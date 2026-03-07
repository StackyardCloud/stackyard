FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /prometheus ./examples/aws/prometheus
ENTRYPOINT ["/prometheus"]
