FROM golang:1.25

WORKDIR /workspace
COPY . .

RUN go run ./examples/azure/computelimit-2026-04-30 >/tmp/compute-limit-example.out

CMD ["go", "run", "./examples/azure/computelimit-2026-04-30"]
