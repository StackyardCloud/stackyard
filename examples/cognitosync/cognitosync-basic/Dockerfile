FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognitosync-basic ./examples/cognitosync/cognitosync-basic
ENTRYPOINT ["/cognitosync-basic"]
