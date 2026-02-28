FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lambda-basic ./examples/lambda/lambda-basic
ENTRYPOINT ["/lambda-basic"]
