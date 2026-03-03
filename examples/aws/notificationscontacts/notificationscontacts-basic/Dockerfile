FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notificationscontacts-basic ./examples/aws/notificationscontacts/notificationscontacts-basic
ENTRYPOINT ["/notificationscontacts-basic"]
