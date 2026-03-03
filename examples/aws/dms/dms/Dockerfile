FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /dms ./examples/aws/dms/dms
ENTRYPOINT ["/dms"]
