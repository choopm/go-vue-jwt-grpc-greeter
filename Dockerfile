FROM golang
ARG CI_PROJECT_DIR=/builds/greeter
ARG CI_PROJECT_URL=https://gitlab.0pointer.org/choopm/greeter
ARG CI_SERVER_HOST=gitlab.0pointer.org
ARG CI_JOB_USER=gitlab-ci-token
ARG CI_JOB_TOKEN=
RUN export REPO_NAME=`echo $CI_PROJECT_URL|sed 's/.*:\/\///g;'` && \
    mkdir -p $GOPATH/src/$(dirname $REPO_NAME) && \
    ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME
WORKDIR $CI_PROJECT_DIR
COPY . .
RUN git config --global url.https://$CI_JOB_USER:$CI_JOB_TOKEN@$CI_SERVER_HOST/.insteadOf https://$CI_SERVER_HOST/ && \
    GOPRIVATE=$CI_SERVER_HOST CGO_ENABLED=0 go mod download
RUN GOPRIVATE=$CI_SERVER_HOST CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /greeter ./cmd/greeter/main.go

FROM alpine
RUN apk add --no-cache openssl
COPY --from=0 /greeter /greeter
COPY docker-entrypoint.sh /
COPY web /web
ENV BEARER_TOKEN "/data/bearer.token"
ENV TLS_CRT "/data/tls.crt"
ENV TLS_KEY "/data/tls.key"
ENV PASSWD "/data/passwd"
ENV ADDRESS "0.0.0.0"
ENV PORT "443"
ENV HTTPS_REDIRECT "true"
VOLUME [ "/data" ]
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/greeter"]
