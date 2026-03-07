FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /shopping-css-apiv1 ./examples/gcp/shopping-css-apiv1

ENTRYPOINT ["/shopping-css-apiv1"]
