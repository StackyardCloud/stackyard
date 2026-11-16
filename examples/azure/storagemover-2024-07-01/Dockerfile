FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storagemover-2024-07-01 ./examples/azure/storagemover-2024-07-01

ENTRYPOINT ["/storagemover-2024-07-01"]
