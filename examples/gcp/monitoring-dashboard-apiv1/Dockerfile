FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /monitoring-dashboard-apiv1 ./examples/gcp/monitoring-dashboard-apiv1

ENTRYPOINT ["/monitoring-dashboard-apiv1"]
