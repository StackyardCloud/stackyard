FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-lowlatency-advanced ./examples/aws/ivs-lowlatency/ivs-lowlatency-advanced
ENTRYPOINT ["/ivs-lowlatency-advanced"]
