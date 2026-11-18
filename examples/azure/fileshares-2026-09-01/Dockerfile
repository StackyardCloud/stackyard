FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /fileshares-2026-09-01 ./examples/azure/fileshares-2026-09-01

ENTRYPOINT ["/fileshares-2026-09-01"]
