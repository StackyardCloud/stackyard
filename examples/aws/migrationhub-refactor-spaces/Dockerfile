FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-refactor-spaces ./examples/aws/migrationhub-refactor-spaces
ENTRYPOINT ["/migrationhub-refactor-spaces"]
