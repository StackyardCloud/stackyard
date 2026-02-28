FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recyclebin-advanced ./examples/recyclebin/recyclebin-advanced
ENTRYPOINT ["/recyclebin-advanced"]
