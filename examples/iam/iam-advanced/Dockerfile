FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iam-advanced ./examples/iam/iam-advanced
ENTRYPOINT ["/iam-advanced"]
