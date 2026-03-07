FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /usernotifications ./examples/aws/usernotifications
ENTRYPOINT ["/usernotifications"]
