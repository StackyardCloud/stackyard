FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /transcribe-basic ./examples/transcribe/transcribe-basic
ENTRYPOINT ["/transcribe-basic"]
