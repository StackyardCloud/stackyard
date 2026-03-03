FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediaconnect-basic ./examples/aws/mediaconnect/mediaconnect-basic
ENTRYPOINT ["/mediaconnect-basic"]
