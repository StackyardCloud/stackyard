FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /recoveryservices-2024-04-01 ./examples/azure/recoveryservices-2024-04-01

ENTRYPOINT ["/recoveryservices-2024-04-01"]
