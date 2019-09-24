FROM golang:1.12

WORKDIR /app

COPY ./go.mod ./go.sum /app/
RUN go get

COPY ./main.go  /app/main.go
RUN go build -o answer
CMD ["/app/answer"]

EXPOSE 50000

