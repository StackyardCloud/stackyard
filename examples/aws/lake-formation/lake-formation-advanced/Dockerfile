FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lake-formation-advanced ./examples/aws/lake-formation/lake-formation-advanced
ENTRYPOINT ["/lake-formation-advanced"]
