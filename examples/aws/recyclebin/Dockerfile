FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /recyclebin ./examples/aws/recyclebin
ENTRYPOINT ["/recyclebin"]
