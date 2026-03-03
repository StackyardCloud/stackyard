FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-refactor-spaces-advanced ./examples/aws/migrationhub-refactor-spaces/migrationhub-refactor-spaces-advanced
ENTRYPOINT ["/migrationhub-refactor-spaces-advanced"]
