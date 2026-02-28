FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rekognition-advanced ./examples/rekognition/rekognition-advanced
ENTRYPOINT ["/rekognition-advanced"]
