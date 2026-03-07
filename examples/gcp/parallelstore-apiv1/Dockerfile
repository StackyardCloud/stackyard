FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /parallelstore-apiv1 ./examples/gcp/parallelstore-apiv1

ENTRYPOINT ["/parallelstore-apiv1"]
