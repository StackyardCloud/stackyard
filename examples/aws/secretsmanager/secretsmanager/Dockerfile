FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /secretsmanager ./examples/aws/secretsmanager/secretsmanager
ENTRYPOINT ["/secretsmanager"]
