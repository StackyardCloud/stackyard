FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /functions-apiv1 ./examples/gcp/functions-apiv1

ENTRYPOINT ["/functions-apiv1"]
