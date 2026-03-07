FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /resourcemanager-apiv3 ./examples/gcp/resourcemanager-apiv3

ENTRYPOINT ["/resourcemanager-apiv3"]
