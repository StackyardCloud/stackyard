FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /shell-apiv1 ./examples/gcp/shell-apiv1

ENTRYPOINT ["/shell-apiv1"]
