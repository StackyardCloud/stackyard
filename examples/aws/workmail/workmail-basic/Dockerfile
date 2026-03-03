FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workmail-basic ./examples/aws/workmail/workmail-basic
ENTRYPOINT ["/workmail-basic"]
