FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediaconnect ./examples/aws/mediaconnect/mediaconnect
ENTRYPOINT ["/mediaconnect"]
