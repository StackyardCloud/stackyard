FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lambda-advanced ./examples/lambda/lambda-advanced
ENTRYPOINT ["/lambda-advanced"]
