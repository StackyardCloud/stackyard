FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-accounts-apiv1 ./examples/gcp/shopping-merchant-accounts-apiv1

ENTRYPOINT ["/shopping-merchant-accounts-apiv1"]
