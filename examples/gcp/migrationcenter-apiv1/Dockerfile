FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /migrationcenter-apiv1 ./examples/gcp/migrationcenter-apiv1

ENTRYPOINT ["/migrationcenter-apiv1"]
