FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chatmessaging ./examples/aws/ivs-chatmessaging
ENTRYPOINT ["/ivs-chatmessaging"]
