FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-zonal-shift-advanced ./examples/aws/arc-zonal-shift/arc-zonal-shift-advanced
ENTRYPOINT ["/arc-zonal-shift-advanced"]
