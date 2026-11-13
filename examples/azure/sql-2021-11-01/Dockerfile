FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /sql-2021-11-01 ./examples/azure/sql-2021-11-01

ENTRYPOINT ["/sql-2021-11-01"]
