FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /netapp-apiv1 ./examples/gcp/netapp-apiv1

ENTRYPOINT ["/netapp-apiv1"]
