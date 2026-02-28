FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notifications-advanced ./examples/notifications/notifications-advanced
ENTRYPOINT ["/notifications-advanced"]
