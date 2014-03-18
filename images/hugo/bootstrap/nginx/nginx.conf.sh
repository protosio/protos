#!/bin/bash

cat <<END
server {
  listen        80;
  server_name   www.$website_name $website_name;
  error_log     /home/www/logs/$website_name.error.log;
  error_page    404    /404.html;

  location / {
    autoindex on;
    root  /home/www/$website_name/public;
  }

}
END
