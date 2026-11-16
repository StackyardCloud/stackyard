FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /managedservices-2022-10-01 ./examples/azure/managedservices-2022-10-01

ENTRYPOINT ["/managedservices-2022-10-01"]
