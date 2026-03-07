FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /networksecurity-apiv1beta1 ./examples/gcp/networksecurity-apiv1beta1

ENTRYPOINT ["/networksecurity-apiv1beta1"]
