FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /frontdoorservice-webapplicationfirewall-2025-11-01 ./examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01

ENTRYPOINT ["/frontdoorservice-webapplicationfirewall-2025-11-01"]
