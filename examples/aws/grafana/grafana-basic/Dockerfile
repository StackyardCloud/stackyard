FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /grafana-basic ./examples/aws/grafana/grafana-basic
ENTRYPOINT ["/grafana-basic"]
