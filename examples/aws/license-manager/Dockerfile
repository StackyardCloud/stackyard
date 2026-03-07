FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager ./examples/aws/license-manager
ENTRYPOINT ["/license-manager"]
