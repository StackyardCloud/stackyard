FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /monitoring-dashboard-apiv1 ./examples/gcp/monitoring-dashboard-apiv1

ENTRYPOINT ["/monitoring-dashboard-apiv1"]
