FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /codecatalyst ./examples/aws/codecatalyst
ENTRYPOINT ["/codecatalyst"]
