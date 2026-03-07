FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /lustre-apiv1 ./examples/gcp/lustre-apiv1

ENTRYPOINT ["/lustre-apiv1"]
