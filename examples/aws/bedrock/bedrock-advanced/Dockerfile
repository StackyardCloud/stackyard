FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-advanced ./examples/aws/bedrock/bedrock-advanced
ENTRYPOINT ["/bedrock-advanced"]
