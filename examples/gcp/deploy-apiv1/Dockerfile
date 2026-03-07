FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /deploy-apiv1 ./examples/gcp/deploy-apiv1

ENTRYPOINT ["/deploy-apiv1"]
