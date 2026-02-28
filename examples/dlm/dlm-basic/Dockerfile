FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /dlm-basic ./examples/dlm/dlm-basic
CMD ["/dlm-basic"]
