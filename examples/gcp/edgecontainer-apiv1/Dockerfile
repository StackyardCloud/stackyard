FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /edgecontainer-apiv1 ./examples/gcp/edgecontainer-apiv1

ENTRYPOINT ["/edgecontainer-apiv1"]
