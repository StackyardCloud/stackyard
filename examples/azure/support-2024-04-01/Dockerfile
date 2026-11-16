FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /support-2024-04-01 ./examples/azure/support-2024-04-01

ENTRYPOINT ["/support-2024-04-01"]
