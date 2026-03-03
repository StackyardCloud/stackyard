FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend ./examples/aws/comprehend/comprehend
ENTRYPOINT ["/comprehend"]
