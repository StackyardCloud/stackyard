FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /eventgrid-namespace-2024-06-01-example ./examples/azure/eventgrid-namespace-2024-06-01

ENTRYPOINT ["/eventgrid-namespace-2024-06-01-example"]
