FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /migrate-discovery-2020-01-01 ./examples/azure/migrate-discovery-2020-01-01

ENTRYPOINT ["/migrate-discovery-2020-01-01"]
