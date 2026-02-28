FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /eks-advanced ./examples/eks/eks-advanced
ENTRYPOINT ["/eks-advanced"]
