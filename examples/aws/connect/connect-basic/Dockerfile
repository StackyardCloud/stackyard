FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /connect-basic ./examples/aws/connect/connect-basic
ENTRYPOINT ["/connect-basic"]
