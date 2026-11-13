FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /digitaltwins-2023-01-31 ./examples/azure/digitaltwins-2023-01-31

ENTRYPOINT ["/digitaltwins-2023-01-31"]
