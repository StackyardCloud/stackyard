FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /rekognition ./examples/aws/rekognition
ENTRYPOINT ["/rekognition"]
