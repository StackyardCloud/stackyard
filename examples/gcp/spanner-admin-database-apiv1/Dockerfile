FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /spanner-admin-database-apiv1 ./examples/gcp/spanner-admin-database-apiv1

ENTRYPOINT ["/spanner-admin-database-apiv1"]
