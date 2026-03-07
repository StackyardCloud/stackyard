FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /vision-apiv1 ./examples/gcp/vision-apiv1

ENTRYPOINT ["/vision-apiv1"]
