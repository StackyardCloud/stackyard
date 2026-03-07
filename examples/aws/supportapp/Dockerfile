FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /supportapp ./examples/aws/supportapp
ENTRYPOINT ["/supportapp"]
