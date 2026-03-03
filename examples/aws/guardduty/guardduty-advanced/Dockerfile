FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /guardduty-advanced ./examples/aws/guardduty/guardduty-advanced
ENTRYPOINT ["/guardduty-advanced"]
