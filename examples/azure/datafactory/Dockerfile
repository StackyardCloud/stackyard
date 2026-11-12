FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datafactory-example ./examples/azure/datafactory

ENTRYPOINT ["/datafactory-example"]
