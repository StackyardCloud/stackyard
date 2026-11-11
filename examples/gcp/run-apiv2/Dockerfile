FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /run-apiv2 ./examples/gcp/run-apiv2

ENTRYPOINT ["/run-apiv2"]
