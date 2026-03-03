FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chatmessaging ./examples/aws/ivs-chatmessaging/ivs-chatmessaging
ENTRYPOINT ["/ivs-chatmessaging"]
