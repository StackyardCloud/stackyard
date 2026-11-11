FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /dms ./examples/aws/dms
ENTRYPOINT ["/dms"]
