FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /sagemaker-basic ./examples/sagemaker/sagemaker-basic
ENTRYPOINT ["/sagemaker-basic"]
