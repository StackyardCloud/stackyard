FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storageinsights-apiv1 ./examples/gcp/storageinsights-apiv1

ENTRYPOINT ["/storageinsights-apiv1"]
