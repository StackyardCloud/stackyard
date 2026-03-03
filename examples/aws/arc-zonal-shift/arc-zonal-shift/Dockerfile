FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-zonal-shift ./examples/aws/arc-zonal-shift/arc-zonal-shift
ENTRYPOINT ["/arc-zonal-shift"]
