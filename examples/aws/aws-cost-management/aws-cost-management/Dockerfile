FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-cost-management ./examples/aws/aws-cost-management/aws-cost-management
ENTRYPOINT ["/aws-cost-management"]
