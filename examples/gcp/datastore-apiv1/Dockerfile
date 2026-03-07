FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /datastore-apiv1 ./examples/gcp/datastore-apiv1

ENTRYPOINT ["/datastore-apiv1"]
