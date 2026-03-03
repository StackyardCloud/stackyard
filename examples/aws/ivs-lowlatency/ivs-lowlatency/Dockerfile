FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-lowlatency ./examples/aws/ivs-lowlatency/ivs-lowlatency
ENTRYPOINT ["/ivs-lowlatency"]
