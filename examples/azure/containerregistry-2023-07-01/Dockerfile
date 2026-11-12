FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /containerregistry-2023-07-01 ./examples/azure/containerregistry-2023-07-01

ENTRYPOINT ["/containerregistry-2023-07-01"]
