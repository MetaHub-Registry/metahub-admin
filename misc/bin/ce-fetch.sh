#!/bin/sh
set -x

if [[ -z "$1" ]]; then
    tmp_dir=$(mktemp -d -t ce-XXXXXXXXXX)
    echo "${tmp_dir}" |tee .ce-tmpdir.txt
else
    rm -rf $1/*
    tmp_dir=$1
fi
if [[ -z "$CE_VER" ]]; then
    CE_VER=0.0.7
fi
wget -qO - https://gitlab.com/qnib-metahub/community-edition/-/archive/v${CE_VER}/community-edition-v${CE_VER}.tar.gz \
    |tar xfz - -C ${tmp_dir} --strip-components=1

touch ${tmp_dir}/.done