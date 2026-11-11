FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /chaosstudio-2025-01-01 ./examples/azure/chaosstudio-2025-01-01

ENTRYPOINT ["/chaosstudio-2025-01-01"]
