FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /notificationscontacts ./examples/aws/notificationscontacts
ENTRYPOINT ["/notificationscontacts"]
