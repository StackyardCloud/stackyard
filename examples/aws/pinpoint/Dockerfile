FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /pinpoint ./examples/aws/pinpoint
ENTRYPOINT ["/pinpoint"]
