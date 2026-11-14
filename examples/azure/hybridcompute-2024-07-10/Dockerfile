FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /hybridcompute-2024-07-10 ./examples/azure/hybridcompute-2024-07-10

ENTRYPOINT ["/hybridcompute-2024-07-10"]
