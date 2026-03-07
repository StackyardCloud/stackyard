FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chatmessaging ./examples/aws/ivs-chatmessaging
ENTRYPOINT ["/ivs-chatmessaging"]
