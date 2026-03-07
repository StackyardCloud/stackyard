FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager-user-subscriptions ./examples/aws/license-manager-user-subscriptions
ENTRYPOINT ["/license-manager-user-subscriptions"]
