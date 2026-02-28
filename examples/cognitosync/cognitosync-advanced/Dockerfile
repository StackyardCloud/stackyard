FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognitosync-advanced ./examples/cognitosync/cognitosync-advanced
ENTRYPOINT ["/cognitosync-advanced"]
