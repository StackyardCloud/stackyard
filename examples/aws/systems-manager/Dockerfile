FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /systems-manager ./examples/aws/systems-manager
ENTRYPOINT ["/systems-manager"]
