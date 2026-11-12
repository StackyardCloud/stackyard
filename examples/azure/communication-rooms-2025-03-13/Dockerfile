FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-rooms-2025-03-13 ./examples/azure/communication-rooms-2025-03-13

ENTRYPOINT ["/communication-rooms-2025-03-13"]
