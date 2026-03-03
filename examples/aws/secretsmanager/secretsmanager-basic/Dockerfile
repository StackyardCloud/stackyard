FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /secretsmanager-basic ./examples/aws/secretsmanager/secretsmanager-basic
ENTRYPOINT ["/secretsmanager-basic"]
