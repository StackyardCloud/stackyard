FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codecatalyst-basic ./examples/codecatalyst/codecatalyst-basic
ENTRYPOINT ["/codecatalyst-basic"]
