# Body Rewrite

Body Rewrite plugin middleware modifies a response by replacing one specified string by another.

## Configuration

To configure this plugin you should add its configuration to the Traefik dynamic configuration as explained [here](https://docs.traefik.io/getting-started/configuration-overview/#the-dynamic-configuration).
The following snippet shows how to configure this plugin with the File provider in TOML and YAML: 


Static:

```toml
[experimental.pilot]
  token = "xxxx"
[experimental.plugins.tomato]
  modulename = "github.com/containous/plugin-rewritebody"
  version = "v0.1.0"
```

Dynamic:

```toml
  [http.middlewares]
    [http.middlewares.my-rewritebody.plugin.rewritebody]
        [[http.middlewares.my-rewritebody.plugin.rewritebody.rewrites]]
            regex = "foo"
            replacement = "bar"

        [[http.middlewares.my-rewritebody.plugin.rewritebody.rewrites]]
            regex = "bar"
            replacement = "foobar"
```

```yaml
http:
  middlewares:
    my-rewritebody:
      plugin:  
        rewritebody:
          rewrites:
            - regex: "foo"
              replacement: "bar"
            - regex: "bar"
              replacement: "foobar"
```
