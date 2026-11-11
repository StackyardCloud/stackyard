FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /guestconfiguration-2024-04-05 ./examples/azure/guestconfiguration-2024-04-05

ENTRYPOINT ["/guestconfiguration-2024-04-05"]
