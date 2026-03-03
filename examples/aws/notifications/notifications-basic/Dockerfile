FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notifications-basic ./examples/aws/notifications/notifications-basic
ENTRYPOINT ["/notifications-basic"]
