FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager-linux-subscriptions-advanced ./examples/license-manager-linux-subscriptions/license-manager-linux-subscriptions-advanced
ENTRYPOINT ["/license-manager-linux-subscriptions-advanced"]
