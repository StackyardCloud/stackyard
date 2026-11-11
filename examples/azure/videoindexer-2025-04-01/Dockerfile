FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /videoindexer-2025-04-01 ./examples/azure/videoindexer-2025-04-01

ENTRYPOINT ["/videoindexer-2025-04-01"]
