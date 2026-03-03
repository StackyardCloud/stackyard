FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager-user-subscriptions-advanced ./examples/aws/license-manager-user-subscriptions/license-manager-user-subscriptions-advanced
ENTRYPOINT ["/license-manager-user-subscriptions-advanced"]
