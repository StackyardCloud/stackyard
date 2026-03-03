FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /pinpoint-basic ./examples/aws/pinpoint/pinpoint-basic
ENTRYPOINT ["/pinpoint-basic"]
