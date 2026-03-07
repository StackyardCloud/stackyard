FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storagetransfer-apiv1 ./examples/gcp/storagetransfer-apiv1

ENTRYPOINT ["/storagetransfer-apiv1"]
