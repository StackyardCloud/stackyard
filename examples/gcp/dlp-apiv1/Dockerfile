FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dlp-apiv1 ./examples/gcp/dlp-apiv1

ENTRYPOINT ["/dlp-apiv1"]
