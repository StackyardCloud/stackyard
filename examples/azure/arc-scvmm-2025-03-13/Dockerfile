FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /arc-scvmm-2025-03-13 ./examples/azure/arc-scvmm-2025-03-13

ENTRYPOINT ["/arc-scvmm-2025-03-13"]
