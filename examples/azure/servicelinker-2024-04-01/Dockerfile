FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /servicelinker-2024-04-01 ./examples/azure/servicelinker-2024-04-01

ENTRYPOINT ["/servicelinker-2024-04-01"]
