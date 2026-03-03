FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-medical ./examples/aws/comprehend-medical/comprehend-medical
ENTRYPOINT ["/comprehend-medical"]
