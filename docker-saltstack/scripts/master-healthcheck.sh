#!/bin/bash

# Health check for Salt Master
salt-run manage.status > /dev/null 2>&1
