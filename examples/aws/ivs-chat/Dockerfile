FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-chat ./examples/aws/ivs-chat
ENTRYPOINT ["/ivs-chat"]
