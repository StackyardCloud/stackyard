FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /translate ./examples/aws/translate
ENTRYPOINT ["/translate"]
