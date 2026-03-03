FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mpa ./examples/aws/mpa/mpa
ENTRYPOINT ["/mpa"]
