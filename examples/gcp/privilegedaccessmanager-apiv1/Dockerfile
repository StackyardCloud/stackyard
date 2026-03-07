FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /privilegedaccessmanager-apiv1 ./examples/gcp/privilegedaccessmanager-apiv1

ENTRYPOINT ["/privilegedaccessmanager-apiv1"]
