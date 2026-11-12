FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-reports-apiv1 ./examples/gcp/shopping-merchant-reports-apiv1
CMD ["/shopping-merchant-reports-apiv1"]
