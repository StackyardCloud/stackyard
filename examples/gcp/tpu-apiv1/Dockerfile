FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /tpu-apiv1 ./examples/gcp/tpu-apiv1

ENTRYPOINT ["/tpu-apiv1"]
