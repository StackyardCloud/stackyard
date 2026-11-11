FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-cost-management ./examples/aws/aws-cost-management
ENTRYPOINT ["/aws-cost-management"]
