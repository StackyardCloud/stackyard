FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /maintenance-2023-04-01 ./examples/azure/maintenance-2023-04-01

ENTRYPOINT ["/maintenance-2023-04-01"]
