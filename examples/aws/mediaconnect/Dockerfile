FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediaconnect ./examples/aws/mediaconnect
ENTRYPOINT ["/mediaconnect"]
