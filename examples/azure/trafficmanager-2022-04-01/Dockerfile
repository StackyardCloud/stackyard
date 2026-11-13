FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /trafficmanager-2022-04-01 ./examples/azure/trafficmanager-2022-04-01

ENTRYPOINT ["/trafficmanager-2022-04-01"]
