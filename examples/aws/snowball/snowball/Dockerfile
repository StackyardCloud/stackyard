FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /snowball ./examples/aws/snowball/snowball
ENTRYPOINT ["/snowball"]

