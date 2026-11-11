FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /eventarc-publishing-apiv1 ./examples/gcp/eventarc-publishing-apiv1

ENTRYPOINT ["/eventarc-publishing-apiv1"]
