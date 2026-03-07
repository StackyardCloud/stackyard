FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-quota-apiv1 ./examples/gcp/shopping-merchant-quota-apiv1
CMD ["/shopping-merchant-quota-apiv1"]
