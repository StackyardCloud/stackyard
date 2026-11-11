FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /privatecatalog-apiv1beta1 ./examples/gcp/privatecatalog-apiv1beta1

ENTRYPOINT ["/privatecatalog-apiv1beta1"]
