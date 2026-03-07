FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-inventories-apiv1 ./examples/gcp/shopping-merchant-inventories-apiv1
CMD ["/shopping-merchant-inventories-apiv1"]
