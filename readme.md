# **Docker Nginx Auto Proxy**
This docker image automatic proxies requests to your docker containers

## Usage

## Configuration

First, pull the image from docker hub

    # use correct tag
    docker pull stephenafamo/docker-nginx-auto-proxy:4.x.x 

Run a container

    docker run --name nginx -v /path/to/my/config/directory:/docker/config -p 80:80 -p 443:443 stephenafamo/docker-nginx-auto-proxy:4.x.x

The container reads any file with the extension `.toml` in `/docker/config`. You can change this folder using the `CONFIG_DIR` environmental variable.

To easily manage all proxies, you should mount your own configuration directory.
`-v /path/to/my/config/dir:/docker/config`

### Variables

These are the environmental variables you can use to tweak the behaviour of this image.

1. `EMAIL`: The email used to accept the TOS for getting Let's Encrypt certificates. **REQUIRED**
1. `CONFIG_DIR`: This is a set of directories where the container will look for `.config` files. Multiple directories are separated with a colon `:`. Default `/docker/config`.
1. `CONFIG_RELOAD_TIME`: This image automatically checks for changes to your configuration files. This environmental variable is used to set how long it should wait between checks. Default is `5s`. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". Examples of durations are:
    * 5s: 5 seconds
    * 1m: 1 minute
    * 1m30s: 1 minute, 30 seconds
    * 12h: 12 hours
1. `HTTPS_VALIDITY`: How often the entire config should be purged and reconfigured even if there are no changes. This is useful for things like auto-renewing letsencrypt certificates. Default `168h`(1 week).
1. `LETSENCRYPT_CREDS_DIR`: The directory where credential files for `certbot` dns plugins will be placed. Default is `/docker/letsencrypt-credentials`
1. `LETSENCRYPT_DNS_PROPAGATION`: Seconds to wait for dns propagation when using the dns authentication method. Default is `120`


## Writing configuration files

A configuration file is a set of defined services. You can put multiple services in a single file, and you can have multiple files in the `CONFIG_DIR` or any of its subdirectories. All configuration files must end with `.toml`.  Services are defined using the [toml format](https://github.com/toml-lang/toml).

The parameters used to define a service are based on the type of proxy needed. HTTP or TCP/UDP. 

Each service should be the TOML equivalient of the [`ServiceConfig` type](https://github.com/stephenafamo/nginx-proxy-load-balancer/blob/master/internal/types.go#L36).

* The top level `Location`, `LocationOptions`, `Upstream`, and `UpstreamOptions` are an easy way to set only a single key in `Locations` (where `Match` == `Location`). Since many services will define only a single location, it's easier to do

```toml
[unique-key]
domains = ["my.domain.com"]
upstream = [{address = "upstream.io"}]
ssl = true
sslSource = "letsencrypt"
httpsOnly = true
```

than:

```toml
[unique-key]
domains = ["my.domain.com"]
ssl = true
sslSource = "letsencrypt"
httpsOnly = true
[[unique-key.locations]]
upstream = [{address = "upstream.io"}]
match = "/"
```

Both ways are completely valid though.

See comments on the [`ServiceConfig` type](https://github.com/stephenafamo/nginx-proxy-load-balancer/blob/master/internal/types.go#L36). struct for details. Some examples will be added soon (PRs welcome).

## Let's Encrypt

If set up correctly, the container will attempt to get a new certificate if there was none, or renew the certificate.

Certificates that were last configured {`HTTPS_VALIDITY`} ago will automatically be retried.

## Roadmap

* ~~Load balancing with multiple containers~~ **DONE**
* ~~Automatic SSL support with let's encrypt~~ **DONE**
* ~~Allow custom ssl certificate configuration.~~ **DONE

Please inform me of any issues or feature requests, Pull requests are appreciated.
