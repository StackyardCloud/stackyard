FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /healthcareapis-2024-03-31 ./examples/azure/healthcareapis-2024-03-31

ENTRYPOINT ["/healthcareapis-2024-03-31"]
