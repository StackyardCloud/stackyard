FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /social-messaging ./examples/aws/social-messaging
ENTRYPOINT ["/social-messaging"]
