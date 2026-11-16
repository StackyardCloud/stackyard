FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /network-2024-05-01 ./examples/azure/network-2024-05-01

ENTRYPOINT ["/network-2024-05-01"]
