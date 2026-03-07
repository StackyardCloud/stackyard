FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /run-apiv2 ./examples/gcp/run-apiv2

ENTRYPOINT ["/run-apiv2"]
