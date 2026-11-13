FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /deviceprovisioningservices-2022-02-05 ./examples/azure/deviceprovisioningservices-2022-02-05

ENTRYPOINT ["/deviceprovisioningservices-2022-02-05"]
