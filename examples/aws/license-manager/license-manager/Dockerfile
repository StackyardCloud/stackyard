FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager ./examples/aws/license-manager/license-manager
ENTRYPOINT ["/license-manager"]
