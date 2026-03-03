FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /sagemaker-basic ./examples/aws/sagemaker/sagemaker-basic
ENTRYPOINT ["/sagemaker-basic"]
