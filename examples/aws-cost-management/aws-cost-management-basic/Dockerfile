FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-cost-management-basic ./examples/aws-cost-management/aws-cost-management-basic
ENTRYPOINT ["/aws-cost-management-basic"]
