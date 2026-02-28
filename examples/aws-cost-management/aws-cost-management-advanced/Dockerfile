FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-cost-management-advanced ./examples/aws-cost-management/aws-cost-management-advanced
ENTRYPOINT ["/aws-cost-management-advanced"]
