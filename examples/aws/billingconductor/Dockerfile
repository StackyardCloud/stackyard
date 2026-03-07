FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /billingconductor ./examples/aws/billingconductor
ENTRYPOINT ["/billingconductor"]
