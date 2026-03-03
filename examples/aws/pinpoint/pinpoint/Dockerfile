FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /pinpoint ./examples/aws/pinpoint/pinpoint
ENTRYPOINT ["/pinpoint"]
