FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /securityhub-basic ./examples/securityhub/securityhub-basic
ENTRYPOINT ["/securityhub-basic"]
