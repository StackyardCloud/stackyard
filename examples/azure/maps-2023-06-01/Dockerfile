FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-2023-06-01 ./examples/azure/maps-2023-06-01

ENTRYPOINT ["/maps-2023-06-01"]
