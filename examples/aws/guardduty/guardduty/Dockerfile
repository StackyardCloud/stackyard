FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /guardduty ./examples/aws/guardduty/guardduty
ENTRYPOINT ["/guardduty"]
