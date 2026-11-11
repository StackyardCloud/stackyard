FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /webpubsub-controlplane-2024-03-01 ./examples/azure/webpubsub-controlplane-2024-03-01

ENTRYPOINT ["/webpubsub-controlplane-2024-03-01"]
