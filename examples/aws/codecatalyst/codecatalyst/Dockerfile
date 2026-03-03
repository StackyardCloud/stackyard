FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codecatalyst ./examples/aws/codecatalyst/codecatalyst
ENTRYPOINT ["/codecatalyst"]
