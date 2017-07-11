# **Docker Nginx Auto Proxy**
This docker image automatic proxies requests to your docker containers

## Usage


1. Pull the docker image using `docker pull stephenafamo/docker-nginx-auto-proxy`.
2. Write your domain configuration
3. Mount the configuration to `/docker-config/nginx.config` e.g `docker run --name nginx-proxy -v /path/to/my/config.ext:/docker-config/nginx.config -d stephenafamo/docker-nginx-auto-proxy`
3. All configuration should be in a single file of the following format

    `config.conf`

        container_name="domains.to.be.proxied"="optional/sub/directory"

    for example

        awesome_stephen="example.com www.example.com *.example.com"
        awesome_blog="stephenafamo.com"="blog"

    **NOTE**

    * Domain should not include the scheme `http://`
    * The sub_directory should not start with the forward slash `/`
    * If the target container cannot be reached, no configuration will be created. If the target container is in network, make sure that you add the nginx container to it too.

## Simple configuration

You can use the environmental variables `CONTAINER`, `DOMAIN` and `DIR` to set a single line of the config while bringing up the container

    docker run --name nginx-proxy \
    -e CONTAINER=awesome_blog \
    -e DOMAIN="stephenafamo.com" \
    -e DIR="blog" \
    -p 80:80 \
    -d stephenafamo/docker-nginx-auto-proxy

## Additional commands 

The following commands are available through the contianer.

1. **active_domains**: Will list out the domains that have been configured
2. **add_config container domain path**: Will add a domain to the config file
3. **load_config**: Will re-generate configuration files and reload nginx

## Roadmap

* Load balancing with multiple containers
* Automatic SSL support with let's encrypt

Please inform me of any issues, Pull requests are appreciated.
