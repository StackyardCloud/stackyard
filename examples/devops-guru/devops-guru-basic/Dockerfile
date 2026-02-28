FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /devops-guru-basic ./examples/devops-guru/devops-guru-basic
ENTRYPOINT ["/devops-guru-basic"]
