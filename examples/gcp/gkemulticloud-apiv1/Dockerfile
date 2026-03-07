FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /gkemulticloud-apiv1 ./examples/gcp/gkemulticloud-apiv1

ENTRYPOINT ["/gkemulticloud-apiv1"]
