FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datafusion-apiv1 ./examples/gcp/datafusion-apiv1

ENTRYPOINT ["/datafusion-apiv1"]
