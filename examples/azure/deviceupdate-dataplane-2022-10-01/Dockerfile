FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /deviceupdate-dataplane-2022-10-01-example ./examples/azure/deviceupdate-dataplane-2022-10-01

ENTRYPOINT ["/deviceupdate-dataplane-2022-10-01-example"]
