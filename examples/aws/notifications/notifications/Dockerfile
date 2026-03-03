FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notifications ./examples/aws/notifications/notifications
ENTRYPOINT ["/notifications"]
