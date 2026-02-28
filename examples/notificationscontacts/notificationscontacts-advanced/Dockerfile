FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /notificationscontacts-advanced ./examples/notificationscontacts/notificationscontacts-advanced
ENTRYPOINT ["/notificationscontacts-advanced"]
