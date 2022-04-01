#!/bin/bash

# Generates docs/reference_pkg.html files for each package, slightly nicer
# than what godoc generates by default. Requires having godoc installed.

headhtml="<!DOCTYPE html><html><head><title>etxt/godoc</title><link href="https://godoc.org/-/bootstrap.min.css" rel="stylesheet"><style>body { max-width: 840px; margin: 20px auto; font-size: 16px; }</style></head><body>"
tailhtml="</body></html>"
packages="emask efixed esizer ecache etxt"

# main etxt package docs
godoc -html . > tmp_doc_body
echo "$headhtml" > tmp_doc_head
echo "$tailhtml" > tmp_doc_tail
cat tmp_doc_head tmp_doc_body tmp_doc_tail > docs/reference_etxt.html
for pkg in $packages; do
	sed -i "s|/pkg/github.com/tinne26/etxt/$pkg/|./reference_$pkg.html|g" docs/reference_etxt.html
	sed -i "s|/pkg/golang.org/|https://pkg.go.dev/golang.org/|g" docs/reference_etxt.html
done

# subpackage docs
for pkg in $packages; do
	if [[ "$pkg" != 'etxt' ]]; then
		godoc -html "./$pkg" > tmp_doc_body
		echo "$headhtml" > tmp_doc_head
		echo "$tailhtml" > tmp_doc_tail
		cat tmp_doc_head tmp_doc_body tmp_doc_tail > "docs/reference_$pkg.html"
		for pkg in $packages; do
			sed -i "s|/pkg/github.com/tinne26/etxt/$pkg/|./reference_$pkg.html|g" "docs/reference_$pkg.html"
			sed -i "s|/pkg/golang.org/|https://pkg.go.dev/golang.org/|g" "docs/reference_$pkg.html"
		done
	fi
done

# clear temp files
rm tmp_doc_head tmp_doc_body tmp_doc_tail

#cp docs/reference_etxt.html docs/reference_emask.html
