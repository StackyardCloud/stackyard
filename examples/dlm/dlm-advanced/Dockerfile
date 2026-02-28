FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /dlm-advanced ./examples/dlm/dlm-advanced
CMD ["/dlm-advanced"]
