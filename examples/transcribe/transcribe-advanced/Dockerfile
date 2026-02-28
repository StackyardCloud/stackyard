FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /transcribe-advanced ./examples/transcribe/transcribe-advanced
ENTRYPOINT ["/transcribe-advanced"]
