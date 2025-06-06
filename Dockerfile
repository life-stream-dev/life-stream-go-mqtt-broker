FROM amd64/golang:1.23-bookworm
LABEL authors="Half_nothing"

WORKDIR /opt/mqtt

ENV GOPROXY https://mirrors.aliyun.com/goproxy/

COPY go.* ./
COPY "cmd/" "./cmd"
COPY internal/ ./internal

RUN go mod download
RUN cd go build -o bin/mqtt-broker ./cmd/mqtt-broker

EXPOSE 9090
EXPOSE 1883

CMD ["./bin/mqtt-broker"]