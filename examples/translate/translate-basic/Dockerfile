FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /translate-basic ./examples/translate/translate-basic
ENTRYPOINT ["/translate-basic"]
