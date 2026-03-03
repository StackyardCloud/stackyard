FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /detective-basic ./examples/aws/detective/detective-basic
ENTRYPOINT ["/detective-basic"]
