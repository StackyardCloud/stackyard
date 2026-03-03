FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /translate-advanced ./examples/aws/translate/translate-advanced
ENTRYPOINT ["/translate-advanced"]
