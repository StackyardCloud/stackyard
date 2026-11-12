FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /supportapp ./examples/aws/supportapp
ENTRYPOINT ["/supportapp"]
