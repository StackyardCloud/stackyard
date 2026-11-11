FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-chat-2025-03-15 ./examples/azure/communication-chat-2025-03-15

ENTRYPOINT ["/communication-chat-2025-03-15"]
