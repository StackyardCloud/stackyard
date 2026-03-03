FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /billingconductor ./examples/aws/billingconductor/billingconductor
ENTRYPOINT ["/billingconductor"]
