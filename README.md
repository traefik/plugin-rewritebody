# Rewrite Body

Rewrite body is a middleware plugin for [Traefik](https://github.com/traefik/traefik) which rewrites the HTTP response body
by replacing a search regex by a replacement string.

## Configuration

### Static

```toml
[pilot]
  token = "xxxx"

[experimental.plugins.rewritebody]
  modulename = "github.com/traefik/plugin-rewritebody"
  version = "v0.3.1"
```

### Dynamic

To configure the `Rewrite Body` plugin you should create a [middleware](https://docs.traefik.io/middlewares/overview/) in 
your dynamic configuration as explained [here](https://docs.traefik.io/middlewares/overview/). The following example creates
and uses the `rewritebody` middleware plugin to replace all foo occurences by bar in the HTTP response body.`

You can replace with variables from the `http.Request` variable. For example to get the `req.Host` variable use `{.Host}` in your
replacement string. You can change the delimiter to use something different that the brackets. See examples for more information.

If you want to apply some limits on the response body, you can chain this middleware plugin with the [Buffering middleware](https://docs.traefik.io/middlewares/buffering/) from Traefik.

```toml
[http.routers]
  [http.routers.my-router]
    rule = "Host(`localhost`)"
    middlewares = ["rewrite-foo"]
    service = "my-service"

[http.middlewares]
  [http.middlewares.rewrite-foo.plugin.rewritebody]
    # Keep Last-Modified header returned by the HTTP service.
    # By default, the Last-Modified header is removed.
    lastModified = true

    # Rewrites all "foo" occurences by "bar"
    [[http.middlewares.rewrite-foo.plugin.rewritebody.rewrites]]
      regex = "foo"
      replacement = "bar"

    # Rewrites all "bar" occurences by the host requested
    [[http.middlewares.rewrite-foo.plugin.rewritebody.rewrites]]
      regex = "bar"
      replacement = "{.Host}"
    
    # Rewrites all "example.com" occurences by the Method requested and by changing the delimiters
    [[http.middlewares.rewrite-foo.plugin.rewritebody.rewrites]]
      regex = "example.com"
      replacement = "/.Method/"
      delimiterLeft = "/"
      delimiterRight = "/"      

[http.services]
  [http.services.my-service]
    [http.services.my-service.loadBalancer]
      [[http.services.my-service.loadBalancer.servers]]
        url = "http://127.0.0.1"
```
