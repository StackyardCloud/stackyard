FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-zonal-shift ./examples/aws/arc-zonal-shift
ENTRYPOINT ["/arc-zonal-shift"]
