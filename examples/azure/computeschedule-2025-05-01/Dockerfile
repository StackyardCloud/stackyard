FROM golang:1.25

WORKDIR /workspace
COPY . .

RUN go run ./examples/azure/computeschedule-2025-05-01 >/tmp/compute-schedule-example.out

CMD ["go", "run", "./examples/azure/computeschedule-2025-05-01"]
