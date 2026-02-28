FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /snowball-advanced ./examples/snowball/snowball-advanced
ENTRYPOINT ["/snowball-advanced"]

