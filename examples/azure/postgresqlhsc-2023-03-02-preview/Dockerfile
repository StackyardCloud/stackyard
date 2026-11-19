FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /postgresqlhsc-2023-03-02-preview ./examples/azure/postgresqlhsc-2023-03-02-preview

ENTRYPOINT ["/postgresqlhsc-2023-03-02-preview"]
