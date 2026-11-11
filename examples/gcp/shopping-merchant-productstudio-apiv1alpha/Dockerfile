FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-productstudio-apiv1alpha ./examples/gcp/shopping-merchant-productstudio-apiv1alpha
CMD ["/shopping-merchant-productstudio-apiv1alpha"]
