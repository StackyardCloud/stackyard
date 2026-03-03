FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /finspace ./examples/aws/finspace/finspace
CMD ["/finspace"]
