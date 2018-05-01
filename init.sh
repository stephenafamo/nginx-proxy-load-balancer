#!/bin/bash

init_db
load_config

# Reload modified files periodically
while true; do
    load_config -m
    sleep `printenv CONFIG_RELOAD_TIME`
done &

nginx -g "daemon off;"
