FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /privilegedaccessmanager-apiv1 ./examples/gcp/privilegedaccessmanager-apiv1

ENTRYPOINT ["/privilegedaccessmanager-apiv1"]
