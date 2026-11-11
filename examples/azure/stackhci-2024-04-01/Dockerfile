FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /stackhci-2024-04-01 ./examples/azure/stackhci-2024-04-01

ENTRYPOINT ["/stackhci-2024-04-01"]
