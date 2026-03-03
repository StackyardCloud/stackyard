FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager-linux-subscriptions-basic ./examples/aws/license-manager-linux-subscriptions/license-manager-linux-subscriptions-basic
ENTRYPOINT ["/license-manager-linux-subscriptions-basic"]
