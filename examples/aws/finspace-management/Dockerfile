FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /finspace-management ./examples/aws/finspace-management
CMD ["/finspace-management"]
