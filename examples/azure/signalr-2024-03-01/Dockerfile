FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /signalr-2024-03-01 ./examples/azure/signalr-2024-03-01

ENTRYPOINT ["/signalr-2024-03-01"]
