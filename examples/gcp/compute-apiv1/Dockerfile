FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /compute-apiv1 ./examples/gcp/compute-apiv1

ENTRYPOINT ["/compute-apiv1"]
