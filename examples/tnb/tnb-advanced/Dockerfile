FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /tnb-advanced ./examples/tnb/tnb-advanced
ENTRYPOINT ["/tnb-advanced"]
