FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /security-privateca-apiv1 ./examples/gcp/security-privateca-apiv1

ENTRYPOINT ["/security-privateca-apiv1"]
