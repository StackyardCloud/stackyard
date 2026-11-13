FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /monitor-2023-01-01 ./examples/azure/monitor-2023-01-01

ENTRYPOINT ["/monitor-2023-01-01"]
