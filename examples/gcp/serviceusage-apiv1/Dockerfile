FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /serviceusage-apiv1 ./examples/gcp/serviceusage-apiv1

ENTRYPOINT ["/serviceusage-apiv1"]
