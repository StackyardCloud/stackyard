FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /finspace-advanced ./examples/finspace/finspace-advanced
CMD ["/finspace-advanced"]
