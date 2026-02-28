FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /secretsmanager-advanced ./examples/secretsmanager/secretsmanager-advanced
ENTRYPOINT ["/secretsmanager-advanced"]
