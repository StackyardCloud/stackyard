FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lake-formation-basic ./examples/lake-formation/lake-formation-basic
ENTRYPOINT ["/lake-formation-basic"]
