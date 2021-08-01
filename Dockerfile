FROM golang:alpine
ARG CI_PROJECT_DIR=/builds/greeter
ARG CI_PROJECT_URL=https://gitlab.0pointer.org/choopm/greeter
ARG CI_SERVER_HOST=gitlab.0pointer.org
ARG CI_JOB_USER=gitlab-ci-token
ARG CI_JOB_TOKEN=
RUN apk add --no-cache git gcc musl-dev
RUN export REPO_NAME=`echo $CI_PROJECT_URL|sed 's/.*:\/\///g;'` && \
    mkdir -p $GOPATH/src/$(dirname $REPO_NAME) && \
    ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME
WORKDIR $CI_PROJECT_DIR
COPY go.* .
RUN git config --global url.https://$CI_JOB_USER:$CI_JOB_TOKEN@$CI_SERVER_HOST/.insteadOf https://$CI_SERVER_HOST/ && \
    GOPRIVATE=$CI_SERVER_HOST go mod download
COPY . .
RUN GOPRIVATE=$CI_SERVER_HOST go build -a -o /greeter ./cmd/greeter/main.go

FROM node:alpine
ARG CI_PROJECT_DIR=/builds/greeter
WORKDIR $CI_PROJECT_DIR
COPY package.json .
COPY yarn.lock .
RUN yarn install
COPY . .
RUN yarn build

FROM alpine
RUN apk add --no-cache openssl libcap
RUN set -eux; \
	addgroup -S -g 900 appuser; \
	adduser -S -u 900 -G appuser -s /bin/sh -h / -H -D appuser;
COPY --from=0 /greeter /greeter
COPY --from=1 /builds/greeter/dist /static
COPY docker-entrypoint.sh /
RUN setcap 'cap_net_bind_service=+ep' /greeter
USER appuser
ENV DATABASE "/data/gorm.db"
ENV TLS_CRT "/data/tls.crt"
ENV TLS_KEY "/data/tls.key"
ENV ADDRESS "0.0.0.0"
ENV PORT "443"
ENV HTTP_REDIRECT "true"
ENV COOKIE_AUTH "false"
VOLUME [ "/data" ]
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/greeter"]
