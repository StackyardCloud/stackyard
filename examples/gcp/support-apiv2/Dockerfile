FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /support-apiv2 ./examples/gcp/support-apiv2

ENTRYPOINT ["/support-apiv2"]
