FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /frontdoorservice-frontdoor-2019-05-01 ./examples/azure/frontdoorservice-frontdoor-2019-05-01

ENTRYPOINT ["/frontdoorservice-frontdoor-2019-05-01"]
