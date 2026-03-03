FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rekognition-basic ./examples/aws/rekognition/rekognition-basic
ENTRYPOINT ["/rekognition-basic"]
