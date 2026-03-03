FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /tnb-basic ./examples/aws/tnb/tnb-basic
ENTRYPOINT ["/tnb-basic"]
