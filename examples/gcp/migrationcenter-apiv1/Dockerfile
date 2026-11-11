FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /migrationcenter-apiv1 ./examples/gcp/migrationcenter-apiv1

ENTRYPOINT ["/migrationcenter-apiv1"]
