FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-promotions-apiv1 ./examples/gcp/shopping-merchant-promotions-apiv1
CMD ["/shopping-merchant-promotions-apiv1"]
