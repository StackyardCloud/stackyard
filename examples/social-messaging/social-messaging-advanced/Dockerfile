FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /social-messaging-advanced ./examples/social-messaging/social-messaging-advanced
ENTRYPOINT ["/social-messaging-advanced"]
