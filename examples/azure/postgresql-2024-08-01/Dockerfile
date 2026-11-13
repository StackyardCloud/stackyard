FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /postgresql-2024-08-01 ./examples/azure/postgresql-2024-08-01

ENTRYPOINT ["/postgresql-2024-08-01"]
