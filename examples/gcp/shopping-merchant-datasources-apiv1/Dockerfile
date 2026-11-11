FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-datasources-apiv1 ./examples/gcp/shopping-merchant-datasources-apiv1
CMD ["/shopping-merchant-datasources-apiv1"]
