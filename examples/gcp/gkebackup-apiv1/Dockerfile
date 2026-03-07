FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /gkebackup-apiv1 ./examples/gcp/gkebackup-apiv1

ENTRYPOINT ["/gkebackup-apiv1"]
