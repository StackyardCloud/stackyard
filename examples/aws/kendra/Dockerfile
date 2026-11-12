FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /kendra ./examples/aws/kendra
ENTRYPOINT ["/kendra"]
