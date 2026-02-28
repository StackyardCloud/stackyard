FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resource-explorer-basic ./examples/resource-explorer/resource-explorer-basic
ENTRYPOINT ["/resource-explorer-basic"]
