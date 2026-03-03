FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-refactor-spaces ./examples/aws/migrationhub-refactor-spaces/migrationhub-refactor-spaces
ENTRYPOINT ["/migrationhub-refactor-spaces"]
