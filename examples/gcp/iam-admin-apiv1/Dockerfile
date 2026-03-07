FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /iam-admin-apiv1 ./examples/gcp/iam-admin-apiv1

ENTRYPOINT ["/iam-admin-apiv1"]
