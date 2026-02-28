FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /social-messaging-basic ./examples/social-messaging/social-messaging-basic
ENTRYPOINT ["/social-messaging-basic"]
