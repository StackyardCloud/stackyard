FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /functions-apiv2 ./examples/gcp/functions-apiv2

ENTRYPOINT ["/functions-apiv2"]
