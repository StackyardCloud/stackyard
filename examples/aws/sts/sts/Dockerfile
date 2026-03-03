FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /sts ./examples/aws/sts/sts
ENTRYPOINT ["/sts"]
