FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /blueprints-2018-11-01-preview ./examples/azure/blueprints-2018-11-01-preview

ENTRYPOINT ["/blueprints-2018-11-01-preview"]
