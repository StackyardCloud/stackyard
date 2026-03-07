FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /metricsscope-apiv1 ./examples/gcp/metricsscope-apiv1

ENTRYPOINT ["/metricsscope-apiv1"]
