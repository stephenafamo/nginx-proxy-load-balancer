# **Docker Nginx Auto Proxy**
This docker image automatic proxies requests to your docker containers

## Usage

## Configuration

First, pull the image from docker hub

    docker pull stephenafamo/docker-nginx-auto-proxy:3.0.0

Run a container

    docker run --name nginx -v /path/to/my/config/directory:/docker/config -p 80:80 -p 443:443 stephenafamo/docker-nginx-auto-proxy:4.0.0

The container reads any file with the extension `.toml` in `/docker/config`. You can change this folder using the `CONFIG_DIR` environmental variable.

To easily manage all proxies, you should mount your own configuration directory.
`-v /path/to/my/config/dir:/docker/config`

### Variables

These are the environmental variables you can use to tweak the behaviour of this image.

1. `CONFIG_DIR`: This is a set of directories where the container will look for `.config` files. Multiple directories are separated with a colon `:`. Default `/docker/config`.
2. `CONFIG_RELOAD_TIME`: This image automatically checks for changes to your configuration files. This environmental variable is used to set how long it should wait between checks. Default is `5s`. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". Examples of durations are:
    * 5s: 5 seconds
    * 1m: 1 minute
    * 1m30s: 1 minute, 30 seconds
    * 12h: 12 hours
3. `CONFIG_VALIDITY`: How often the entire config should be purged and reconfigured even if there are no changes. This is useful for things like auto-renewing letsencrypt certificates. Default `604800s`(1 week).
4. `EMAIL`: The email used to accept the TOS for getting Let's Encrypt certificates.


## Writing configuration files

A configuration file is a set of defined services. You can put multiple services in a single file, and you can have multiple files in the `CONFIG_DIR` or any of its subdirectories. All configuration files must end with `.toml`.  Services are defined using the [toml format](https://github.com/toml-lang/toml).

The parameters used to define a service are based on the type of proxy needed. HTTP or TCP/UDP. 

### HTTP Proxy & Load balancing

The syntax is as follows(showing all possible fields).

        "myblog"
        "UPSTREAM"main.stephenafamo.com | 1st.stephenafamo.com weight=3 | 2nd.stephenafamo.com max_fails=3 fail_timeout=30s"
        "UPSTREAM_OPTIONS"ip_hash | keepalive 32"
        "DOMAIN"stephenafamo.com"
        "DIRECTORY"blog"
        "SSL"1"
        "SSL_SOURCE"letsencrypt"
        "HTTPS_ONLY"1"
        "TYPE"HTTP"
        "myblog"


1. The only required fields are `UPSTREAM` and `DOMAIN`
1. A block of configuration should be started and ended by the configuration name. This name should be unique. In the example above, the configuration name is `myblog`
2. Neither the domain or upstream address should include the scheme `http://`
3. `UPSTREAM` must be reachable or the config will not be generated.
4. For load balancing, you can add multiple `UPSTREAM` addresses. Separate them with pipes.
5. You can add any extra parameters at the end of a single upstream server. [Read this](http://nginx.org/en/docs/http/ngx_http_upstream_module.html#server).
6. The `UPSTREAM_OPTIONS` are not required. Use only if you need to add extra directives to the upstream block for fine tuning. Separate directives with pipes. [Read this](http://nginx.org/en/docs/http/ngx_http_upstream_module.html).
7. `DOMAIN` can be multiple, but should be seperated by spaces
8. `DIRECTORY` is the optional path to to be proxies. For example, if you'd like to proxy only `example.com/blog`, the `blog` will be the directory
9. `SSL` can be enable by setting the parameter to `1`
10. `SSL_SOURCE` for now, only letsencrypt is supported. Certificates will be generated automatically. Soon, mannual configuration will be supported. To be able to re-use the generated certificates, you should mount your `/etc/letsencrypt` folder into the container `-v /etc/letsencrypt:/etc/letsencrypt`. THis only works if `SSL` is `1`.
11. `HTTPS_ONLY` If this is set to `1`, then all `http` requests will be redirected to `https`.
12. `TYPE`: For a `http` proxy, you can omit this field or set it to `HTTP`.

### TCP & UDP Load balancing

The syntax is as follows(showing all possible fields).

        "myblog"
        "UPSTREAM"main.stephenafamo.com | 1st.stephenafamo.com weight=3 | 2nd.stephenafamo.com max_fails=3 fail_timeout=30s"
        "UPSTREAM_OPTIONS"ip_hash | keepalive 32"
        "SERVER_OPTIONS"proxy_buffer_size 16k | proxy_download_rate 0"
        "PORT"12345"
        "TYPE"TCP"
        "myblog"


1. The only required fields are `UPSTREAM`, `PORT` and `TYPE`.
1. A block of configuration should be started and ended by the configuration name. This name should be unique. In the example above, the configuration name is `myblog`
2. Neither the domain or upstream address should include the scheme `http://`
3. `UPSTREAM` must be reachable or the config will not be generated.
4. For load balancing, you can add multiple `UPSTREAM` addresses. Separate them with pipes.
5. You can add any extra parameters at the end of a single upstream server. [Read this](http://nginx.org/en/docs/http/ngx_http_upstream_module.html#server).
6. The `UPSTREAM_OPTIONS` are not required. Use only if you need to add extra directives to the **upstream** block for fine tuning. Separate directives with pipes. [Read this](http://nginx.org/en/docs/http/ngx_http_upstream_module.html). Anything with context `upstream` can be used.
7. The `SERVER_OPTIONS` are not required. Use only if you need to add extra directives to the **server** block for fine tuning. Separate directives with pipes. [Read this](http://nginx.org/en/docs/stream/ngx_stream_proxy_module.html). Anything with context `stream, server` can be used.
8. `TYPE`: Can be set to either `TCP` or `UDP` depending on the type of traffic to load balance.
9. `PORT` is the port which nginx should listen on for the `TCP` or `UDP` traffic.

## Additional commands 

The following commands are available through the contianer.

1. **_active_domains_**: Will list out the domains that have been configured
3. **_load_config_**: Will re-generate configuration files and reload nginx

## Let's Encrypt

If set up correctly, the container will attempt to get a new certificate if there was none, or renew the certificate.
The normal `letsencrypt renew` command may fail. Instead, to renew certificates, run the `load_config` command and certificate renewal will be attempted during the process. 
You should set up a cron to do this automatically e.g `30 2 * * 1 docker exec nginx load_config >> /var/log/nginx-reload.log`

## Roadmap

* ~~Load balancing with multiple containers~~ **DONE**
* ~~Automatic SSL support with let's encrypt~~ **DONE**
* Allow custom ssl certificate configuration.

Please inform me of any issues, Pull requests are appreciated.
