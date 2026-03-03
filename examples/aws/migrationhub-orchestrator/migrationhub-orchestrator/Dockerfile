FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-orchestrator ./examples/aws/migrationhub-orchestrator/migrationhub-orchestrator
ENTRYPOINT ["/migrationhub-orchestrator"]
