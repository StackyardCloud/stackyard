FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /marketplacecatalog-2025-05-01 ./examples/azure/marketplacecatalog-2025-05-01

ENTRYPOINT ["/marketplacecatalog-2025-05-01"]
