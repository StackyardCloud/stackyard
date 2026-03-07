FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /iam-credentials-apiv1 ./examples/gcp/iam-credentials-apiv1

ENTRYPOINT ["/iam-credentials-apiv1"]
