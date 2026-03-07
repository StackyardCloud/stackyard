FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storagebatchoperations-apiv1 ./examples/gcp/storagebatchoperations-apiv1

ENTRYPOINT ["/storagebatchoperations-apiv1"]
