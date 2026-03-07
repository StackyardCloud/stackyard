FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /datastore-admin-apiv1 ./examples/gcp/datastore-admin-apiv1

ENTRYPOINT ["/datastore-admin-apiv1"]
