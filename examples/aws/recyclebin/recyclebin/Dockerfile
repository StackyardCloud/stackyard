FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recyclebin ./examples/aws/recyclebin/recyclebin
ENTRYPOINT ["/recyclebin"]
