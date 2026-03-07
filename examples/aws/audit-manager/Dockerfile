FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /audit-manager ./examples/aws/audit-manager
ENTRYPOINT ["/audit-manager"]
