FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /marketplaceordering-2021-01-01 ./examples/azure/marketplaceordering-2021-01-01

ENTRYPOINT ["/marketplaceordering-2021-01-01"]
