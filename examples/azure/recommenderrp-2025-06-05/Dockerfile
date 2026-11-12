FROM golang:1.25

WORKDIR /workspace
COPY . .

RUN go run ./examples/azure/recommenderrp-2025-06-05 >/tmp/compute-recommender-example.out

CMD ["go", "run", "./examples/azure/recommenderrp-2025-06-05"]
