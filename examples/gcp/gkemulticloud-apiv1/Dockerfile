FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /gkemulticloud-apiv1 ./examples/gcp/gkemulticloud-apiv1

ENTRYPOINT ["/gkemulticloud-apiv1"]
