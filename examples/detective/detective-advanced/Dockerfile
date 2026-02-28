FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /detective-advanced ./examples/detective/detective-advanced
ENTRYPOINT ["/detective-advanced"]
