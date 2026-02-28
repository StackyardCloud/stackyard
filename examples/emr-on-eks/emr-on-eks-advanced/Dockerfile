FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /emr-on-eks-advanced ./examples/emr-on-eks/emr-on-eks-advanced
ENTRYPOINT ["/emr-on-eks-advanced"]
