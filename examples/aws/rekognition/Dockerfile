FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /rekognition ./examples/aws/rekognition
ENTRYPOINT ["/rekognition"]
