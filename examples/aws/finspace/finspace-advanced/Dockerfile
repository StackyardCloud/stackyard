FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /finspace-advanced ./examples/aws/finspace/finspace-advanced
CMD ["/finspace-advanced"]
