FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /health-advanced ./examples/health/health-advanced
ENTRYPOINT ["/health-advanced"]
