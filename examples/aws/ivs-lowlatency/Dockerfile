FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-lowlatency ./examples/aws/ivs-lowlatency
ENTRYPOINT ["/ivs-lowlatency"]
