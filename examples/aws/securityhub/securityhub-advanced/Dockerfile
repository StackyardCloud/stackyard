FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /securityhub-advanced ./examples/aws/securityhub/securityhub-advanced
ENTRYPOINT ["/securityhub-advanced"]
