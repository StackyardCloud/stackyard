FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /speech-apiv2 ./examples/gcp/speech-apiv2

ENTRYPOINT ["/speech-apiv2"]
