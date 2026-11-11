FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /domains-apiv1 ./examples/gcp/domains-apiv1

ENTRYPOINT ["/domains-apiv1"]
