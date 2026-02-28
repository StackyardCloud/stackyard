FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /finspace-basic ./examples/finspace/finspace-basic
CMD ["/finspace-basic"]
