FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /iam-apiv3 ./examples/gcp/iam-apiv3

ENTRYPOINT ["/iam-apiv3"]
