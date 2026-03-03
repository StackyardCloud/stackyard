FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /audit-manager ./examples/aws/audit-manager/audit-manager
ENTRYPOINT ["/audit-manager"]
