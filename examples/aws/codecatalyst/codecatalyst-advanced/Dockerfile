FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codecatalyst-advanced ./examples/aws/codecatalyst/codecatalyst-advanced
ENTRYPOINT ["/codecatalyst-advanced"]
