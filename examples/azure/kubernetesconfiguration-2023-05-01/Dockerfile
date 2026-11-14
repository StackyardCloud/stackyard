FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /kubernetesconfiguration-2023-05-01 ./examples/azure/kubernetesconfiguration-2023-05-01

ENTRYPOINT ["/kubernetesconfiguration-2023-05-01"]
