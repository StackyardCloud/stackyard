FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediaconnect-advanced ./examples/mediaconnect/mediaconnect-advanced
ENTRYPOINT ["/mediaconnect-advanced"]
