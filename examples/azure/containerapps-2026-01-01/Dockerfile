FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /containerapps-2026-01-01 ./examples/azure/containerapps-2026-01-01

ENTRYPOINT ["/containerapps-2026-01-01"]
