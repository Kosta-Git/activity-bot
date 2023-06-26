#!/bin/bash

for file in ./abi/*.json; do
  pkg=$(basename "${file%.*}")
  pkg_name="$(echo "$pkg" | awk '{print tolower(substr($0,1,1))substr($0,2)}')"
  out_file="./pkg/abi/$pkg_name/$pkg_name.go"
  echo "Processing $file ..."
  mkdir -p "./pkg/abi/$pkg_name/"
  abigen --abi "$file" --pkg "$pkg_name" --out "$out_file"
done