FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /workflow-executions-apiv1 ./examples/gcp/workflow-executions-apiv1

ENTRYPOINT ["/workflow-executions-apiv1"]
