FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datareplication-2026-05-01-example ./examples/azure/datareplication-2026-05-01

ENTRYPOINT ["/datareplication-2026-05-01-example"]
