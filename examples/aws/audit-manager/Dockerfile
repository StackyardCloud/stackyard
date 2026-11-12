FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /audit-manager ./examples/aws/audit-manager
ENTRYPOINT ["/audit-manager"]
