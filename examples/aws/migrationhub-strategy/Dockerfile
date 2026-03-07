FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-strategy ./examples/aws/migrationhub-strategy
ENTRYPOINT ["/migrationhub-strategy"]
