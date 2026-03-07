FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /lambda ./examples/aws/lambda
ENTRYPOINT ["/lambda"]
