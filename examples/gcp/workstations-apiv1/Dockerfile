FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /workstations-apiv1 ./examples/gcp/workstations-apiv1

ENTRYPOINT ["/workstations-apiv1"]
