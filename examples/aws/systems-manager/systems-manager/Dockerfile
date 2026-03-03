FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /systems-manager ./examples/aws/systems-manager/systems-manager
ENTRYPOINT ["/systems-manager"]
