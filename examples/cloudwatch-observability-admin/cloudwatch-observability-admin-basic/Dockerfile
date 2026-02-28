FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch-observability-admin-basic ./examples/cloudwatch-observability-admin/cloudwatch-observability-admin-basic
ENTRYPOINT ["/cloudwatch-observability-admin-basic"]
