FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /trace-apiv2 ./examples/gcp/trace-apiv2

ENTRYPOINT ["/trace-apiv2"]
