FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /osconfig-apiv1 ./examples/gcp/osconfig-apiv1

ENTRYPOINT ["/osconfig-apiv1"]
