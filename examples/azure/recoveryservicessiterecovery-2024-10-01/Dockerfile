FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /recoveryservicessiterecovery-2024-10-01 ./examples/azure/recoveryservicessiterecovery-2024-10-01

ENTRYPOINT ["/recoveryservicessiterecovery-2024-10-01"]
