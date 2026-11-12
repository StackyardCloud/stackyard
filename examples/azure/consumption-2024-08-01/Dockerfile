FROM golang:1.25

WORKDIR /workspace
COPY . .

RUN go run ./examples/azure/consumption-2024-08-01 >/tmp/consumption-example.out

CMD ["go", "run", "./examples/azure/consumption-2024-08-01"]
