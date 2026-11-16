FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /costmanagement-2023-11-01 ./examples/azure/costmanagement-2023-11-01

ENTRYPOINT ["/costmanagement-2023-11-01"]
