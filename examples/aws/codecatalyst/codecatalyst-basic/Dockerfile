FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codecatalyst-basic ./examples/aws/codecatalyst/codecatalyst-basic
ENTRYPOINT ["/codecatalyst-basic"]
