FROM golang:1.25

WORKDIR /workspace
COPY . .

RUN go run ./examples/azure/compute-2025-04-01 >/tmp/compute-example.out

CMD ["go", "run", "./examples/azure/compute-2025-04-01"]
