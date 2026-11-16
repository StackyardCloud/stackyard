FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /portal-2020-09-01-preview ./examples/azure/portal-2020-09-01-preview

ENTRYPOINT ["/portal-2020-09-01-preview"]
