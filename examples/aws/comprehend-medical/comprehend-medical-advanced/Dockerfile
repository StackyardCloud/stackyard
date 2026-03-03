FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-medical-advanced ./examples/aws/comprehend-medical/comprehend-medical-advanced
ENTRYPOINT ["/comprehend-medical-advanced"]
