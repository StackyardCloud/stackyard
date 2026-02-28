FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /eks-basic ./examples/eks/eks-basic
ENTRYPOINT ["/eks-basic"]
