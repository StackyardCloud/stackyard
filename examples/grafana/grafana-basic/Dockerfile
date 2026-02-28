FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /grafana-basic ./examples/grafana/grafana-basic
ENTRYPOINT ["/grafana-basic"]
