#!/bin/sh

set -e

make

echo "Running chimney..."

./chimney -config=config.json
