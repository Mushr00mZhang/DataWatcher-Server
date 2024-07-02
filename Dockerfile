FROM golang:1.22-alpine as builder
ENV GOPROXY https://goproxy.cn,direct
ENV GO111MODULE on
ENV CGO_ENABLED 0
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -ldflags="-s -w" -o server .

FROM scratch
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["/app/server"]