FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resource-explorer-advanced ./examples/aws/resource-explorer/resource-explorer-advanced
ENTRYPOINT ["/resource-explorer-advanced"]
