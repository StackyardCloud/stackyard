FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storagesync-2022-09-01 ./examples/azure/storagesync-2022-09-01

ENTRYPOINT ["/storagesync-2022-09-01"]
