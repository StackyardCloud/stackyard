FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notifications-advanced ./examples/aws/notifications/notifications-advanced
ENTRYPOINT ["/notifications-advanced"]
