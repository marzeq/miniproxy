### The whats and the whys

#### The whats

MiniProxy is a wrapper around Go's HTTP proxy - `httputil.NewSingleHostReverseProxy`.

It's main goal is to be extremely simple, simpler than even Caddy, not to mention Nginx.
It has no fancy features, just the bare minimum to get the job done.

#### The whys

Because I can.

Well, besides that, it's also because I realised it's probably easier for me to write my own proxy than to learn how to configure Caddy or Nginx properly.

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
"cgi.example.org" = {
  cgi = "./cgi-bin/app" # execute this CGI script for every request on this host
}
"advanced-cgi.example.org" = {
  cgi = {
    path = "./cgi-bin/app"
    dir = "./cgi-bin" # optional working directory
    args = ["--mode", "prod"] # optional extra argv entries
    env = {
      APP_ENV = "production" # optional extra environment variables
    }
    inherit_env = ["PATH"] # optional host environment variables to pass through
  }
}
"foo.example.org" = {
  host = "localhost:4000" # equivalent to "foo.example.org" = "localhost:4000"
}
"webhook.example.org" = {
  shell_before = [
    "git -C /srv/site pull"
  ] # shell_before can be used on its own and returns an empty 200 OK response
}
# "bar.example.org" = {
#   host = "localhost:400"
#   serve_from = "./static" # invalid, you can't have both host and serve_from, makes no sense
#   cgi = "./cgi-bin/app" # also mutually exclusive with host/serve_from
# }

"secure.example.com" = { # example of TLS configuration
  host = "localhost:5000"
  tls = {
    cert = "/some/cert.pem"
    key = "/some/key.pem"
  }
}
```

Default config path is `/etc/miniproxy/config.mconf`.

### License

MIT

### Roadmap

- [x] Basic reverse proxy functionality
- [x] Wildcard domain support
- [x] Static file serving
- [x] Special 404 and 504 handling
- [x] TLS from files
- [x] Shell commands alongside host/serve_from – quasi-webhooks
- [x] CGI handlers
