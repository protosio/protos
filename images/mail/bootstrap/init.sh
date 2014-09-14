#!/bin/bash

chown mysql:mysql /extdata/mysql -R
chown vmail:vmail /extdata/vmail -R
service mysqld start
service dovecot start
service postfix start

# wait forever like a boss
sleep infinity
