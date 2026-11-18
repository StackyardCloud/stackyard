FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /serialconsole-2024-07-01 ./examples/azure/serialconsole-2024-07-01

ENTRYPOINT ["/serialconsole-2024-07-01"]
