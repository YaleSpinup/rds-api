FROM golang:1.21.3-alpine as gobuff

EXPOSE 3000

ENV GOPROXY=https://proxy.golang.org

RUN apk add --no-cache --upgrade apk-tools \
    && apk add --no-cache bash curl openssl git build-base nodejs npm sqlite sqlite-dev mysql-client vim postgresql libpq postgresql-contrib libc6-compat

# Installing linter
RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh \
    | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# Installing Yarn
RUN npm i -g --no-progress yarn \
    && yarn config set yarn-offline-mirror /npm-packages-offline-cache \
    && yarn config set yarn-offline-mirror-pruning true

# Installing buffalo binary
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest
RUN go install github.com/gobuffalo/buffalo-pop/v3@latest

RUN mkdir /src
WORKDIR /src

FROM gobuff as builder

ARG version=0.0.0
ARG prerelease
ARG githash=""
ARG buildstamp=""

RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
ADD . .
RUN go version
RUN buffalo build --static -o /bin/api --ldflags "-X github.com/YaleSpinup/rds-api/rdsapi.Version=$version -X github.com/YaleSpinup/rds-api/rdsapi.VersionPrerelease=$prerelease -X github.com/YaleSpinup/rds-api/rdsapi.GitHash=$githash -X github.com/YaleSpinup/rds-api/rdsapi.BuildStamp=$buildstamp"


FROM alpine
RUN apk add --no-cache bash ca-certificates

WORKDIR /app/

COPY --from=builder /bin/api ./
RUN chmod 555 ./api

# Install Deco
ARG DECO_VERSION=1.4.1
ARG DECO_OS=linux
ARG DECO_ARCH=amd64
ADD https://github.com/YaleUniversity/deco/releases/download/v${DECO_VERSION}/deco_${DECO_VERSION}_${DECO_OS}_${DECO_ARCH}.tar.gz /usr/local/bin/deco.tar.gz
RUN cd /usr/local/bin && tar -zxvf deco.tar.gz && rm -f deco.tar.gz && chmod 555 deco && /usr/local/bin/deco version

COPY docker/*_config.sh ./
COPY docker/config.deco.json ./config/config.json
RUN chmod 600 ./config/config.json

ENV GO_ENV=production
ENV PORT=8080
ENV ADDR=0.0.0.0
EXPOSE 8080

RUN chown -R nobody:nogroup .
USER nobody

CMD ./import_config.sh && ./api -config ./config/config.json
