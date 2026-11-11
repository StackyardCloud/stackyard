FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /education-2021-12-01-preview-example ./examples/azure/education-2021-12-01-preview

ENTRYPOINT ["/education-2021-12-01-preview-example"]

