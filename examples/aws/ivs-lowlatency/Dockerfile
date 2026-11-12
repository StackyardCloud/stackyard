FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-lowlatency ./examples/aws/ivs-lowlatency
ENTRYPOINT ["/ivs-lowlatency"]
