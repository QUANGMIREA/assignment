FROM golang:1.21.0-alpine3.18

WORKDIR /usersegmentator

COPY go.mod go.sum ./
RUN go mod download

COPY cmd  ./cmd
COPY pkg ./pkg
COPY config  ./config
COPY static  ./static
COPY config  ./config

RUN CGO_ENABLED=0 GOOS=linux go build -o /avito-segmentator ./cmd/usersegmentator

CMD ["/avito-segmentator"]
