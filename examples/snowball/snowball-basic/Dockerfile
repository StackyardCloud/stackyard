FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /snowball-basic ./examples/snowball/snowball-basic
ENTRYPOINT ["/snowball-basic"]

