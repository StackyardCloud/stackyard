FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /recyclebin ./examples/aws/recyclebin
ENTRYPOINT ["/recyclebin"]
