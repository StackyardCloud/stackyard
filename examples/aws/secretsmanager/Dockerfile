FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /secretsmanager ./examples/aws/secretsmanager
ENTRYPOINT ["/secretsmanager"]
