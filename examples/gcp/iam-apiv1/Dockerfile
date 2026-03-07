FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /iam-apiv1 ./examples/gcp/iam-apiv1

ENTRYPOINT ["/iam-apiv1"]
