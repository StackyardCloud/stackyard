FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /social-messaging ./examples/aws/social-messaging/social-messaging
ENTRYPOINT ["/social-messaging"]
