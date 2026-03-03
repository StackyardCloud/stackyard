FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-orchestrator-advanced ./examples/aws/migrationhub-orchestrator/migrationhub-orchestrator-advanced
ENTRYPOINT ["/migrationhub-orchestrator-advanced"]
