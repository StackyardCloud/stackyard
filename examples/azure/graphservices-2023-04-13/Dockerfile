FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /graphservices-2023-04-13 ./examples/azure/graphservices-2023-04-13

ENTRYPOINT ["/graphservices-2023-04-13"]
