FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch-observability-admin-advanced ./examples/aws/cloudwatch-observability-admin/cloudwatch-observability-admin-advanced
ENTRYPOINT ["/cloudwatch-observability-admin-advanced"]
