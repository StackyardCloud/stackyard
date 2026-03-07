FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /retail-apiv2 ./examples/gcp/retail-apiv2

ENTRYPOINT ["/retail-apiv2"]
