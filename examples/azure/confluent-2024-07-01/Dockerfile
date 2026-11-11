FROM golang:1.25

WORKDIR /workspace
COPY . .

RUN go run ./examples/azure/confluent-2024-07-01 >/tmp/confluent-example.out

CMD ["go", "run", "./examples/azure/confluent-2024-07-01"]
