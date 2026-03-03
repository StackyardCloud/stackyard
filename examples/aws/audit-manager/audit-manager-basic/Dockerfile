FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /audit-manager-basic ./examples/aws/audit-manager/audit-manager-basic
ENTRYPOINT ["/audit-manager-basic"]
