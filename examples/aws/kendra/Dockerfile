FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /kendra ./examples/aws/kendra
ENTRYPOINT ["/kendra"]
