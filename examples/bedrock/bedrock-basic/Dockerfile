FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-basic ./examples/bedrock/bedrock-basic
ENTRYPOINT ["/bedrock-basic"]
