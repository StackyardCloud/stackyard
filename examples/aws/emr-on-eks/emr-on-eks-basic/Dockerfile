FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /emr-on-eks-basic ./examples/aws/emr-on-eks/emr-on-eks-basic
ENTRYPOINT ["/emr-on-eks-basic"]
