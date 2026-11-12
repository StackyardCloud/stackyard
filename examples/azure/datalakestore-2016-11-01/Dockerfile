FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datalakestore-example ./examples/azure/datalakestore-2016-11-01

ENTRYPOINT ["/datalakestore-example"]
