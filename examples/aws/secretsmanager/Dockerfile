FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /secretsmanager ./examples/aws/secretsmanager
ENTRYPOINT ["/secretsmanager"]
