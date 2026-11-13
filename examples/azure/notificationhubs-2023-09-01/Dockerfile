FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /notificationhubs-2023-09-01 ./examples/azure/notificationhubs-2023-09-01

ENTRYPOINT ["/notificationhubs-2023-09-01"]
