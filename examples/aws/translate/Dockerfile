FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /translate ./examples/aws/translate
ENTRYPOINT ["/translate"]
