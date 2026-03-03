FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /usernotifications-basic ./examples/aws/usernotifications/usernotifications-basic
ENTRYPOINT ["/usernotifications-basic"]
