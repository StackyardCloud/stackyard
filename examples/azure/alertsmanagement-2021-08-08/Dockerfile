FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /alertsmanagement-2021-08-08 ./examples/azure/alertsmanagement-2021-08-08

ENTRYPOINT ["/alertsmanagement-2021-08-08"]
