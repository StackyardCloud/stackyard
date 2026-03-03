FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticache-basic ./examples/aws/elasticache/elasticache-basic
ENTRYPOINT ["/elasticache-basic"]
