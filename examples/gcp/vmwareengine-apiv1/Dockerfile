FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /vmwareengine-apiv1 ./examples/gcp/vmwareengine-apiv1

ENTRYPOINT ["/vmwareengine-apiv1"]
