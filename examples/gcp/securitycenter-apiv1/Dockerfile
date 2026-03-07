FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /securitycenter-apiv1 ./examples/gcp/securitycenter-apiv1

ENTRYPOINT ["/securitycenter-apiv1"]
