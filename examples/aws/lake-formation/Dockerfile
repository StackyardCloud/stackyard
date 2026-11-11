FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /lake-formation ./examples/aws/lake-formation
ENTRYPOINT ["/lake-formation"]
