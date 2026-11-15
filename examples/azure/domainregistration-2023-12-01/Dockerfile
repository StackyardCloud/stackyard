FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /domainregistration-2023-12-01 ./examples/azure/domainregistration-2023-12-01

ENTRYPOINT ["/domainregistration-2023-12-01"]
