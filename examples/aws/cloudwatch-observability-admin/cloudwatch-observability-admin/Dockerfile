FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch-observability-admin ./examples/aws/cloudwatch-observability-admin/cloudwatch-observability-admin
ENTRYPOINT ["/cloudwatch-observability-admin"]
