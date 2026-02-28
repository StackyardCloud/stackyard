FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rolesanywhere-advanced ./examples/rolesanywhere/rolesanywhere-advanced
CMD ["/rolesanywhere-advanced"]
