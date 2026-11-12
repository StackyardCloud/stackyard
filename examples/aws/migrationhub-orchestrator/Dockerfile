FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-orchestrator ./examples/aws/migrationhub-orchestrator
ENTRYPOINT ["/migrationhub-orchestrator"]
