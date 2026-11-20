FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /oracle-2026-06-01 ./examples/azure/oracle-2026-06-01

ENTRYPOINT ["/oracle-2026-06-01"]
