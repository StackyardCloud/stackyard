FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-notifications-apiv1 ./examples/gcp/shopping-merchant-notifications-apiv1
CMD ["/shopping-merchant-notifications-apiv1"]
