FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /workloads-2023-04-01 ./examples/azure/workloads-2023-04-01

ENTRYPOINT ["/workloads-2023-04-01"]
