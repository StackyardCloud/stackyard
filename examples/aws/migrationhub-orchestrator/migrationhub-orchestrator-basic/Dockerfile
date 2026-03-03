FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-orchestrator-basic ./examples/aws/migrationhub-orchestrator/migrationhub-orchestrator-basic
ENTRYPOINT ["/migrationhub-orchestrator-basic"]
