FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /prometheus-advanced ./examples/prometheus/prometheus-advanced
ENTRYPOINT ["/prometheus-advanced"]
