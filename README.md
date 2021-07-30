# greeter

## Go backend

### Running it for development
```
export JWT_SECRET=data/jwt.secret; export TLS_CRT=data/tls.crt; export TLS_KEY=data/tls.key; export PASSWD=data/passwd; export PORT=4443;
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
