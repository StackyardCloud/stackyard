FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-strategy ./examples/aws/migrationhub-strategy/migrationhub-strategy
ENTRYPOINT ["/migrationhub-strategy"]
