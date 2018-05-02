#!/bin/bash

init_db

# Reload modified files periodically
while true; do
    source load_config -m
    sleep `printenv CONFIG_RELOAD_TIME`
done &

nginx -g "daemon off;"
