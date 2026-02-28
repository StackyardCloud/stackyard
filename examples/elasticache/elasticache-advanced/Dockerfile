FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticache-advanced ./examples/elasticache/elasticache-advanced
ENTRYPOINT ["/elasticache-advanced"]
