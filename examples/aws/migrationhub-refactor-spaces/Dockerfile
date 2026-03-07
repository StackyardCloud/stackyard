FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-refactor-spaces ./examples/aws/migrationhub-refactor-spaces
ENTRYPOINT ["/migrationhub-refactor-spaces"]
