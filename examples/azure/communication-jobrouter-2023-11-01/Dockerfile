FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-jobrouter-2023-11-01 ./examples/azure/communication-jobrouter-2023-11-01

ENTRYPOINT ["/communication-jobrouter-2023-11-01"]
