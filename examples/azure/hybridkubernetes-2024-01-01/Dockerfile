FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /hybridkubernetes-2024-01-01 ./examples/azure/hybridkubernetes-2024-01-01

ENTRYPOINT ["/hybridkubernetes-2024-01-01"]
