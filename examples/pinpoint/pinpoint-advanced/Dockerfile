FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /pinpoint-advanced ./examples/pinpoint/pinpoint-advanced
ENTRYPOINT ["/pinpoint-advanced"]
