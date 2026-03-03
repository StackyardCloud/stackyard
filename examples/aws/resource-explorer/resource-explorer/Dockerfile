FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resource-explorer ./examples/aws/resource-explorer/resource-explorer
ENTRYPOINT ["/resource-explorer"]
