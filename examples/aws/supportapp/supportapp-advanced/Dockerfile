FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /supportapp-advanced ./examples/aws/supportapp/supportapp-advanced
ENTRYPOINT ["/supportapp-advanced"]
