FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /recoveryservicesbackup-2024-10-01 ./examples/azure/recoveryservicesbackup-2024-10-01

ENTRYPOINT ["/recoveryservicesbackup-2024-10-01"]
