FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rekognition ./examples/aws/rekognition/rekognition
ENTRYPOINT ["/rekognition"]
