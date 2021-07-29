module.exports = {
    devServer: {
      port: 8081,
      proxy: 'http://localhost:4443',
      headers: { "Access-Control-Allow-Origin": "*" },
    }
  }
