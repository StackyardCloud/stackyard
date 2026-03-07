FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /financialservices-apiv1 ./examples/gcp/financialservices-apiv1

ENTRYPOINT ["/financialservices-apiv1"]
