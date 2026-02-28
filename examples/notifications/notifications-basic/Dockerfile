FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notifications-basic ./examples/notifications/notifications-basic
ENTRYPOINT ["/notifications-basic"]
