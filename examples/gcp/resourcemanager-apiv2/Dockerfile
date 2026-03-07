FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /resourcemanager-apiv2 ./examples/gcp/resourcemanager-apiv2

ENTRYPOINT ["/resourcemanager-apiv2"]
