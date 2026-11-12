FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /eventgrid-controlplane-2025-02-15-example ./examples/azure/eventgrid-controlplane-2025-02-15

ENTRYPOINT ["/eventgrid-controlplane-2025-02-15-example"]
