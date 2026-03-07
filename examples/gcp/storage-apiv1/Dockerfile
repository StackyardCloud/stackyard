FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storage-apiv1 ./examples/gcp/storage-apiv1

ENTRYPOINT ["/storage-apiv1"]
