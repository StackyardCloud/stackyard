FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /websecurityscanner-apiv1 ./examples/gcp/websecurityscanner-apiv1

ENTRYPOINT ["/websecurityscanner-apiv1"]
