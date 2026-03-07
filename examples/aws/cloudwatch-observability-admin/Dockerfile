FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch-observability-admin ./examples/aws/cloudwatch-observability-admin
ENTRYPOINT ["/cloudwatch-observability-admin"]
