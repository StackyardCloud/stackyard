FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chat-advanced ./examples/ivs-chat/ivs-chat-advanced
ENTRYPOINT ["/ivs-chat-advanced"]
