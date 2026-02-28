FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /finspace-management-basic ./examples/finspace-management/finspace-management-basic
CMD ["/finspace-management-basic"]
