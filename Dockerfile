FROM golang:1.18.3-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
          -ldflags='-w -s -extldflags "-static"' -a \
          -o /bin/app .

FROM scratch
COPY --from=builder /bin/app /app
CMD ["/app"]

EXPOSE 1337
