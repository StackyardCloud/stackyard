FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /prometheus-basic ./examples/aws/prometheus/prometheus-basic
ENTRYPOINT ["/prometheus-basic"]
