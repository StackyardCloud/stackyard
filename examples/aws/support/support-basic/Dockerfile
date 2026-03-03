FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /support-basic ./examples/aws/support/support-basic
ENTRYPOINT ["/support-basic"]
