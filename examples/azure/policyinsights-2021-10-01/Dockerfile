FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /policyinsights-2021-10-01 ./examples/azure/policyinsights-2021-10-01

ENTRYPOINT ["/policyinsights-2021-10-01"]
