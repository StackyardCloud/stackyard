FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /standbypool-2026-04-01 ./examples/azure/standbypool-2026-04-01

ENTRYPOINT ["/standbypool-2026-04-01"]
