FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock ./examples/aws/bedrock/bedrock
ENTRYPOINT ["/bedrock"]
