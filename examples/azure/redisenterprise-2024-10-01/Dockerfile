FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /redisenterprise-2024-10-01 ./examples/azure/redisenterprise-2024-10-01

ENTRYPOINT ["/redisenterprise-2024-10-01"]
