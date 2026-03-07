FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /health ./examples/aws/health
ENTRYPOINT ["/health"]
