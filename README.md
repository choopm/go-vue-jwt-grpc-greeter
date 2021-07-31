# greeter

## Go backend

### Running it for development
You need to generate a certificate first, take a look at [docker-entrypoint.sh](docker-entrypoint.sh).

```
export TLS_CRT=data/tls.crt; export TLS_KEY=data/tls.key; export PORT=4443; export DATABASE=data/gorm.db
go run cmd/greeter/main.go
```
## Vue frontend
### Project setup
```
yarn install
```

#### Compiles and hot-reloads for development
```
yarn serve
```

#### Compiles and minifies for production
```
yarn build
```

#### Lints and fixes files
```
yarn lint
```

#### Customize configuration
See [Configuration Reference](https://cli.vuejs.org/config/).
