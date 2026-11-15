FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /iotcentral-2021-06-01 ./examples/azure/iotcentral-2021-06-01

ENTRYPOINT ["/iotcentral-2021-06-01"]
