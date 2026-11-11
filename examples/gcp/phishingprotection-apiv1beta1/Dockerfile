FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /phishingprotection-apiv1beta1 ./examples/gcp/phishingprotection-apiv1beta1

ENTRYPOINT ["/phishingprotection-apiv1beta1"]
