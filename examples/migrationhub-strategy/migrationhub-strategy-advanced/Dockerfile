FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-strategy-advanced ./examples/migrationhub-strategy/migrationhub-strategy-advanced
ENTRYPOINT ["/migrationhub-strategy-advanced"]
