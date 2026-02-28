FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /devops-guru-advanced ./examples/devops-guru/devops-guru-advanced
ENTRYPOINT ["/devops-guru-advanced"]
