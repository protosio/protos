#!/bin/bash
BASE_DIR=/tmp/bootstrap/
HUGO_VERSION=0.10
WEBSITE_NAME=giurgiu.io

# making sure script aborts if a command fails
set -e

# INSTALL PKG's
apt-get update
apt-get install git nginx -y

# Setup environment
mkdir /home/www/$WEBSITE_NAME/data -p
mkdir /home/www/logs -p
chmod +x $BASE_DIR/nginx/nginx.conf.sh && website_name=$WEBSITE_NAME $BASE_DIR/nginx/nginx.conf.sh > /etc/nginx/sites-enabled/$WEBSITE_NAME.conf
cp $BASE_DIR/hugo_$HUGO_VERSION /usr/bin/hugo && chmod +x /usr/bin/hugo
