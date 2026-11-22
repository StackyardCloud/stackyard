FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /codecommit ./examples/aws/codecommit
ENTRYPOINT ["/codecommit"]
