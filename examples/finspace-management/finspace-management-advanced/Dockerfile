FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /finspace-management-advanced ./examples/finspace-management/finspace-management-advanced
CMD ["/finspace-management-advanced"]
