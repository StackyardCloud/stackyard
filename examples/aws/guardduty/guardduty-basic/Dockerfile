FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /guardduty-basic ./examples/aws/guardduty/guardduty-basic
ENTRYPOINT ["/guardduty-basic"]
