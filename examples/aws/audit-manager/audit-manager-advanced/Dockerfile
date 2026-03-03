FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /audit-manager-advanced ./examples/aws/audit-manager/audit-manager-advanced
ENTRYPOINT ["/audit-manager-advanced"]
