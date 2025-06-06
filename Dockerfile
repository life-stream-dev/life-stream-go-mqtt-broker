FROM amd64/golang:1.23-bookworm
LABEL authors="Half_nothing"

WORKDIR /opt/mqtt

ENV GOPROXY https://mirrors.aliyun.com/goproxy/

COPY go.* ./
COPY "cmd/*" "./cmd"
COPY internal/* ./internal

RUN go get && go build -o bin/mqtt-broker ./cmd/mqtt-broker

RUN ./bin/mqtt-broker

ENTRYPOINT ["top", "-b"]