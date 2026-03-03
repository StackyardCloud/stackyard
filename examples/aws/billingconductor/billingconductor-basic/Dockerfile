FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /billingconductor-basic ./examples/aws/billingconductor/billingconductor-basic
ENTRYPOINT ["/billingconductor-basic"]
