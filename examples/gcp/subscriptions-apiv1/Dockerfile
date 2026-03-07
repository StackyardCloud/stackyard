FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /subscriptions-apiv1 ./examples/gcp/subscriptions-apiv1
CMD ["/subscriptions-apiv1"]
