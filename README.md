### Building

```sh
go build ./cmd/miniproxy
```

### Running

```sh
./miniproxy
PORT=8080 ./miniproxy # custom port, 80 by default
CONFIG_FILE=./config.mconf ./miniproxy # custom config file
# of course you can combine them
```

### Configuration

Syntax is mconf, see [this repository](https://github.com/marzeq/mconf) for details.

```mconf
"example.com" = "localhost:3000" # simple domain to address mapping
"*.example.com" = "localhost:3001" # wildcard domain mapping !! IMPORTANT: example.com won't be matched here !!
"api.*.example.com" = "localhost:3002" # more complex wildcard mapping
"example.*" = "localhost:3003" # wildcard in TLD
"*.example.*" = "localhost:3004" # mixing wildcards
"*" = "localhost:3005" # catch-all mapping, will match anything not matched before
"404" = "localhost:3006" # special case, will be used when no other mapping matched
"504" = "localhost:3007" # special case, will be used when upstream is not reachable

"example.org" = {
  serve_from = "./static" # serve static files from this directory
}
"foo.example.org" = {
  host = "localhost:4000" # equivalent to "foo.example.org" = "localhost:4000"
}
# "bar.example.org" = {
#   host = "localhost:400"
#   serve_from = "./static" # invalid, you can't have both host and serve_from, makes no sense
# }
```

Default config path is `/etc/miniproxy/config.mconf`.

### License

MIT
