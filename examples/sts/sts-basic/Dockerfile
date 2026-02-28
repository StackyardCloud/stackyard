FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /sts-basic ./examples/sts/sts-basic
ENTRYPOINT ["/sts-basic"]
