FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mgn-advanced ./examples/aws/mgn/mgn-advanced
ENTRYPOINT ["/mgn-advanced"]
