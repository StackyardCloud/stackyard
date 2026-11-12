FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /frontdoorservice-azurefrontdoorstandardpremium-2025-04-15 ./examples/azure/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15

ENTRYPOINT ["/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15"]
