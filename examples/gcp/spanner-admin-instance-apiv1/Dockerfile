FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /spanner-admin-instance-apiv1 ./examples/gcp/spanner-admin-instance-apiv1

ENTRYPOINT ["/spanner-admin-instance-apiv1"]
