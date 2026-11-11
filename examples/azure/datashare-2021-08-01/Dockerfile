FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datashare-2021-08-01-example ./examples/azure/datashare-2021-08-01

ENTRYPOINT ["/datashare-2021-08-01-example"]
