FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /metastore-apiv1 ./examples/gcp/metastore-apiv1

ENTRYPOINT ["/metastore-apiv1"]
