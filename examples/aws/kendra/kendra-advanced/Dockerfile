FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kendra-advanced ./examples/aws/kendra/kendra-advanced
ENTRYPOINT ["/kendra-advanced"]
