FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /mysql-2023-12-30 ./examples/azure/mysql-2023-12-30

ENTRYPOINT ["/mysql-2023-12-30"]
