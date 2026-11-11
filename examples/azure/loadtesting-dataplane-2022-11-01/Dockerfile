FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /loadtesting-dataplane-2022-11-01 ./examples/azure/loadtesting-dataplane-2022-11-01

ENTRYPOINT ["/loadtesting-dataplane-2022-11-01"]
