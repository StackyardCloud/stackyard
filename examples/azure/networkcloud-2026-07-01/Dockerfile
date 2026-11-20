FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /networkcloud-2026-07-01 ./examples/azure/networkcloud-2026-07-01

ENTRYPOINT ["/networkcloud-2026-07-01"]
