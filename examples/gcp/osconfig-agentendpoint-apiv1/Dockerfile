FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /osconfig-agentendpoint-apiv1 ./examples/gcp/osconfig-agentendpoint-apiv1

ENTRYPOINT ["/osconfig-agentendpoint-apiv1"]
