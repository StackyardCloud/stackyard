FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chat ./examples/aws/ivs-chat/ivs-chat
ENTRYPOINT ["/ivs-chat"]
