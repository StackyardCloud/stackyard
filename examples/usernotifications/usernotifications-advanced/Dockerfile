FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /usernotifications-advanced ./examples/usernotifications/usernotifications-advanced
ENTRYPOINT ["/usernotifications-advanced"]
