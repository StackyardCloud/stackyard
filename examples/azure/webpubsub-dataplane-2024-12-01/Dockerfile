FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /webpubsub-dataplane-2024-12-01 ./examples/azure/webpubsub-dataplane-2024-12-01

ENTRYPOINT ["/webpubsub-dataplane-2024-12-01"]
