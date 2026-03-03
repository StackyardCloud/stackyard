FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recyclebin-basic ./examples/aws/recyclebin/recyclebin-basic
ENTRYPOINT ["/recyclebin-basic"]
