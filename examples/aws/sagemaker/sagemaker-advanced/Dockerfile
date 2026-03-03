FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /sagemaker-advanced ./examples/aws/sagemaker/sagemaker-advanced
ENTRYPOINT ["/sagemaker-advanced"]
