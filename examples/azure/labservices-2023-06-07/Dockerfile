FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /labservices-2023-06-07 ./examples/azure/labservices-2023-06-07

ENTRYPOINT ["/labservices-2023-06-07"]
