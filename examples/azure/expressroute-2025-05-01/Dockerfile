FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /expressroute-2025-05-01-example ./examples/azure/expressroute-2025-05-01

ENTRYPOINT ["/expressroute-2025-05-01-example"]
