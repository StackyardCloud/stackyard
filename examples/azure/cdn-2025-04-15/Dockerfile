FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /cdn-2025-04-15 ./examples/azure/cdn-2025-04-15

ENTRYPOINT ["/cdn-2025-04-15"]
