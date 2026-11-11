FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-products-apiv1 ./examples/gcp/shopping-merchant-products-apiv1
CMD ["/shopping-merchant-products-apiv1"]
