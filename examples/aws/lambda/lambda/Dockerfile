FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lambda ./examples/aws/lambda/lambda
ENTRYPOINT ["/lambda"]
