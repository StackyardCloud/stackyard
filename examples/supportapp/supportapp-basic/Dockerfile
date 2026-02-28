FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /supportapp-basic ./examples/supportapp/supportapp-basic
ENTRYPOINT ["/supportapp-basic"]
