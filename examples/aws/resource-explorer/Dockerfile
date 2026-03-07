FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /resource-explorer ./examples/aws/resource-explorer
ENTRYPOINT ["/resource-explorer"]
