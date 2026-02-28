FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /billingconductor-advanced ./examples/billingconductor/billingconductor-advanced
ENTRYPOINT ["/billingconductor-advanced"]
