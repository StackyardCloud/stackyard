FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chatmessaging-advanced ./examples/ivs-chatmessaging/ivs-chatmessaging-advanced
ENTRYPOINT ["/ivs-chatmessaging-advanced"]
