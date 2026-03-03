FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /health ./examples/aws/health/health
ENTRYPOINT ["/health"]
