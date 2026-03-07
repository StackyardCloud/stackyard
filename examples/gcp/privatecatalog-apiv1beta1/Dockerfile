FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /privatecatalog-apiv1beta1 ./examples/gcp/privatecatalog-apiv1beta1

ENTRYPOINT ["/privatecatalog-apiv1beta1"]
