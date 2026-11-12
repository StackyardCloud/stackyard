FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /avs-2025-09-01 ./examples/azure/avs-2025-09-01

ENTRYPOINT ["/avs-2025-09-01"]
