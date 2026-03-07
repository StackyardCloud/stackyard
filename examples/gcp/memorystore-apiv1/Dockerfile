FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /memorystore-apiv1 ./examples/gcp/memorystore-apiv1

ENTRYPOINT ["/memorystore-apiv1"]
