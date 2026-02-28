FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /guardduty-basic ./examples/guardduty/guardduty-basic
ENTRYPOINT ["/guardduty-basic"]
