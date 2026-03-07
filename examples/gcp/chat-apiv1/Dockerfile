FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /chat-apiv1 ./examples/gcp/chat-apiv1
CMD ["/chat-apiv1"]
