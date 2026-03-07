FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /speech-apiv1 ./examples/gcp/speech-apiv1

ENTRYPOINT ["/speech-apiv1"]
