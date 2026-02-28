FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chatmessaging-basic ./examples/ivs-chatmessaging/ivs-chatmessaging-basic
ENTRYPOINT ["/ivs-chatmessaging-basic"]
