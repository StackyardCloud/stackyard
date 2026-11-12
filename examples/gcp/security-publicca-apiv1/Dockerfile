FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /security-publicca-apiv1 ./examples/gcp/security-publicca-apiv1

ENTRYPOINT ["/security-publicca-apiv1"]
