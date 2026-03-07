FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /securitycentermanagement-apiv1 ./examples/gcp/securitycentermanagement-apiv1

ENTRYPOINT ["/securitycentermanagement-apiv1"]
