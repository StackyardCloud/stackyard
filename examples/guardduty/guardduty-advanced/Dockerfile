FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /guardduty-advanced ./examples/guardduty/guardduty-advanced
ENTRYPOINT ["/guardduty-advanced"]
