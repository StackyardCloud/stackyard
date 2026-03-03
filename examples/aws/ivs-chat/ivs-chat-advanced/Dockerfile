FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chat-advanced ./examples/aws/ivs-chat/ivs-chat-advanced
ENTRYPOINT ["/ivs-chat-advanced"]
