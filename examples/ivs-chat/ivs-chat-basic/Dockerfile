FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chat-basic ./examples/ivs-chat/ivs-chat-basic
ENTRYPOINT ["/ivs-chat-basic"]
