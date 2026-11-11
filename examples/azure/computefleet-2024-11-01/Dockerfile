FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /computefleet-2024-11-01 ./examples/azure/computefleet-2024-11-01

ENTRYPOINT ["/computefleet-2024-11-01"]
