FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /deviceupdate-resourcemanager-2023-07-01-example ./examples/azure/deviceupdate-resourcemanager-2023-07-01

ENTRYPOINT ["/deviceupdate-resourcemanager-2023-07-01-example"]
