FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /securityhub ./examples/aws/securityhub/securityhub
ENTRYPOINT ["/securityhub"]
