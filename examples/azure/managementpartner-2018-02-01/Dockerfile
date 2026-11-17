FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /managementpartner-2018-02-01 ./examples/azure/managementpartner-2018-02-01

ENTRYPOINT ["/managementpartner-2018-02-01"]
