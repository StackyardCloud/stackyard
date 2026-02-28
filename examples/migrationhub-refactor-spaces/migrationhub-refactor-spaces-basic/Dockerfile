FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /migrationhub-refactor-spaces-basic ./examples/migrationhub-refactor-spaces/migrationhub-refactor-spaces-basic
ENTRYPOINT ["/migrationhub-refactor-spaces-basic"]
