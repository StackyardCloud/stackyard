FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /filestore-apiv1 ./examples/gcp/filestore-apiv1

ENTRYPOINT ["/filestore-apiv1"]
