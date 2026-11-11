FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dtl-2018-09-15-example ./examples/azure/dtl-2018-09-15

ENTRYPOINT ["/dtl-2018-09-15-example"]
