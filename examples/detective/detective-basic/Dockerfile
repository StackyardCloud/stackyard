FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /detective-basic ./examples/detective/detective-basic
ENTRYPOINT ["/detective-basic"]
