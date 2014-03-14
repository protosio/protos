#!/bin/bash
BASE_DIR=/tmp/bootstrap/
HOME=/home/git
USERNAME=username
PUBKEY=publickey.pub

# making sure script aborts if a command fails
set -e

# INSTALL PKG's
apt-get update
apt-get install git openssh-server apt-utils -y

# Setup environment
mkdir /var/run/sshd
adduser --gecos "" --disabled-password git
cd $HOME
git clone git://github.com/sitaramc/gitolite
mkdir -p $HOME/bin
gitolite/install -to $HOME/bin
mkdir $HOME/.ssh && mv $BASE_DIR/$PUBKEY $HOME/.ssh/$USERNAME.pub
$HOME/bin/gitolite setup -pk $HOME/.ssh/$USERNAME.pub
chown git:git \.* * -R && chgrp root .ssh && chmod 700 .ssh && chmod 644 .ssh/$USERNAME.pub
locale-gen en_US.UTF-8 en_US
update-locale en_US.UTF-8 en_US
