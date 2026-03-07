FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /texttospeech-apiv1 ./examples/gcp/texttospeech-apiv1

ENTRYPOINT ["/texttospeech-apiv1"]
