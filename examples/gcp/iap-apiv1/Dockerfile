FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /iap-apiv1 ./examples/gcp/iap-apiv1

ENTRYPOINT ["/iap-apiv1"]
