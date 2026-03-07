FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /vision-v2-apiv1 ./examples/gcp/vision-v2-apiv1

ENTRYPOINT ["/vision-v2-apiv1"]
