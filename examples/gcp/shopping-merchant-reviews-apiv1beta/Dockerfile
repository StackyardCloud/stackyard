FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-reviews-apiv1beta ./examples/gcp/shopping-merchant-reviews-apiv1beta
CMD ["/shopping-merchant-reviews-apiv1beta"]
