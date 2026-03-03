FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mgn ./examples/aws/mgn/mgn
ENTRYPOINT ["/mgn"]
