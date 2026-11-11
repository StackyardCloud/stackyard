FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /edgecontainer-apiv1 ./examples/gcp/edgecontainer-apiv1

ENTRYPOINT ["/edgecontainer-apiv1"]
