FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /secretsmanager-advanced ./examples/aws/secretsmanager/secretsmanager-advanced
ENTRYPOINT ["/secretsmanager-advanced"]
