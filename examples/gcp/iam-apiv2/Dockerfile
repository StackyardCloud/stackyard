FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /iam-apiv2 ./examples/gcp/iam-apiv2

ENTRYPOINT ["/iam-apiv2"]
