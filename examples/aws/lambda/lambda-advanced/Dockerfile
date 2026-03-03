FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lambda-advanced ./examples/aws/lambda/lambda-advanced
ENTRYPOINT ["/lambda-advanced"]
