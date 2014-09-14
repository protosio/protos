#!/bin/bash

service mysqld start
service dovecot start
service postfix start

# wait forever like a boss
cat
