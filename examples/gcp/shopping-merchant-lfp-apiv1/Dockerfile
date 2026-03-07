FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-lfp-apiv1 ./examples/gcp/shopping-merchant-lfp-apiv1
CMD ["/shopping-merchant-lfp-apiv1"]
