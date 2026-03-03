FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notificationscontacts ./examples/aws/notificationscontacts/notificationscontacts
ENTRYPOINT ["/notificationscontacts"]
