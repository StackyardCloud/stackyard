FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /wellarchitected-basic ./examples/aws/wellarchitected/wellarchitected-basic
ENTRYPOINT ["/wellarchitected-basic"]
