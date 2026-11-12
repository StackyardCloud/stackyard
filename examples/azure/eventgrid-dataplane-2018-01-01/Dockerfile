FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /eventgrid-dataplane-2018-01-01-example ./examples/azure/eventgrid-dataplane-2018-01-01

ENTRYPOINT ["/eventgrid-dataplane-2018-01-01-example"]
