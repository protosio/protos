#!/usr/bin/env bash

WG_GO_BINARY=wireguard-go
WG_DIR=/var/run/wireguard
PROGRAM="${0##*/}"

create_tun() {
    WG_TUN_NAME_FILE=$WG_DIR/$1.name /usr/local/bin/$WG_GO_BINARY utun
    echo `cat $WG_DIR/$1.name`
}

remove_tun() {
    WG_IFACE=$1
    TUN_IFACE=`cat $WG_DIR/$1.name`
    rm -f $WG_DIR/$TUN_IFACE.sock && rm -f $WG_DIR/$1.name
}

check_uid() {
    [[ $UID == 0 ]] || echo "Please run this script as root"
    exit 1
}

help() {
        echo "Usage: $PROGRAM [ up | down ] INTERFACE"
}


if [[ $# -eq 1 && ( $1 == --help || $1 == -h || $1 == help ) ]]; then
        help
elif [[ $# -eq 2 && $1 == up ]]; then
        check_uid
        create_tun $2
elif [[ $# -eq 2 && $1 == down ]]; then
        check_uid
        remove_tun $2
else
        help
        exit 1
fi

exit 0