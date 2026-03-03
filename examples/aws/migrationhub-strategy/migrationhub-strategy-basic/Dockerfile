FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-strategy-basic ./examples/aws/migrationhub-strategy/migrationhub-strategy-basic
ENTRYPOINT ["/migrationhub-strategy-basic"]
