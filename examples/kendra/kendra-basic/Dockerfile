FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kendra-basic ./examples/kendra/kendra-basic
ENTRYPOINT ["/kendra-basic"]
