FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /hybridnetwork-2023-09-01 ./examples/azure/hybridnetwork-2023-09-01

ENTRYPOINT ["/hybridnetwork-2023-09-01"]
