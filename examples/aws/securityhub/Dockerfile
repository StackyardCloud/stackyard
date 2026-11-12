FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /securityhub ./examples/aws/securityhub
ENTRYPOINT ["/securityhub"]
