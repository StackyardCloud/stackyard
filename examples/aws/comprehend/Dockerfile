FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend ./examples/aws/comprehend
ENTRYPOINT ["/comprehend"]
