FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager-linux-subscriptions ./examples/aws/license-manager-linux-subscriptions/license-manager-linux-subscriptions
ENTRYPOINT ["/license-manager-linux-subscriptions"]
