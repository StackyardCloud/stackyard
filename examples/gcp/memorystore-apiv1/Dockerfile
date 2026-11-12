FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /memorystore-apiv1 ./examples/gcp/memorystore-apiv1

ENTRYPOINT ["/memorystore-apiv1"]
