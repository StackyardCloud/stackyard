FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediaconnect-basic ./examples/mediaconnect/mediaconnect-basic
ENTRYPOINT ["/mediaconnect-basic"]
