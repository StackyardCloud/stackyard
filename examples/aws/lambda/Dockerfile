FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /lambda ./examples/aws/lambda
ENTRYPOINT ["/lambda"]
