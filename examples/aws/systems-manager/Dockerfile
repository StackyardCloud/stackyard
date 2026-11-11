FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /systems-manager ./examples/aws/systems-manager
ENTRYPOINT ["/systems-manager"]
