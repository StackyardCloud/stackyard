FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /supportapp ./examples/aws/supportapp/supportapp
ENTRYPOINT ["/supportapp"]
