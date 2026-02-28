FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-zonal-shift-basic ./examples/arc-zonal-shift/arc-zonal-shift-basic
ENTRYPOINT ["/arc-zonal-shift-basic"]
