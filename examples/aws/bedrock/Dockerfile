FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock ./examples/aws/bedrock
ENTRYPOINT ["/bedrock"]
