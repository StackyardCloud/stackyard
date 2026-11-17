FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /databox-2025-07-01 ./examples/azure/databox-2025-07-01

ENTRYPOINT ["/databox-2025-07-01"]
