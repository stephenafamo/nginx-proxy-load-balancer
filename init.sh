#!/bin/bash

if [[ ! -z "$CONTAINER" ]] && [[ ! -z "$DOMAIN" ]] ; then
	container=$CONTAINER
	domain=$DOMAIN
	directory=""
	if [[ ! -z "$DIR" ]]; then
		directory="$DIR"
	fi
	add_config $container $domain $directory
fi

load_config

nginx -g "daemon off;"
