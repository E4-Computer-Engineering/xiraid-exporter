#!/bin/sh
set -e

systemctl stop xiraid_exporter.service || true
systemctl disable xiraid_exporter.service || true

systemctl daemon-reload
