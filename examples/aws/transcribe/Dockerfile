FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /transcribe ./examples/aws/transcribe
ENTRYPOINT ["/transcribe"]
