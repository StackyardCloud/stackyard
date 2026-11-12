FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /defenderforcloud-2025-06-01-example ./examples/azure/defenderforcloud-2025-06-01

ENTRYPOINT ["/defenderforcloud-2025-06-01-example"]
