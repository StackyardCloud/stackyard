FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /grafana-advanced ./examples/aws/grafana/grafana-advanced
ENTRYPOINT ["/grafana-advanced"]
