FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /eventarc-apiv1 ./examples/gcp/eventarc-apiv1

ENTRYPOINT ["/eventarc-apiv1"]
