FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /horizondb-2026-10-01-preview ./examples/azure/horizondb-2026-10-01-preview

ENTRYPOINT ["/horizondb-2026-10-01-preview"]
