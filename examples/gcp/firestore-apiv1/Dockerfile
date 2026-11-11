FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /firestore-apiv1 ./examples/gcp/firestore-apiv1

ENTRYPOINT ["/firestore-apiv1"]
