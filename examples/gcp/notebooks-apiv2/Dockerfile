FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /notebooks-apiv2 ./examples/gcp/notebooks-apiv2

ENTRYPOINT ["/notebooks-apiv2"]
