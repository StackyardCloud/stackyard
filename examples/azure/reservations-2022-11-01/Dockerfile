FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /reservations-2022-11-01 ./examples/azure/reservations-2022-11-01

ENTRYPOINT ["/reservations-2022-11-01"]
