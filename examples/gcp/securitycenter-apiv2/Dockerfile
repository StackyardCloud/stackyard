FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /securitycenter-apiv2 ./examples/gcp/securitycenter-apiv2

ENTRYPOINT ["/securitycenter-apiv2"]
