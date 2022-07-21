#!/bin/bash

# Generates docs/reference_pkg.html files for each package, slightly nicer
# than what godoc generates by default. Requires having godoc installed.

# You can open the results after running the script with:
# >> xdg-open docs/reference_etxt.html

pkgname="etxt"
pkgurl="github.com/tinne26/$pkgname"
subpkgs="emask efixed esizer ecache"
docsfolder="docs/"
docsprefix="reference_"

headhtml="<!DOCTYPE html><html><head><title>$pkgurl/godoc</title><link href="https://godoc.org/-/bootstrap.min.css" rel="stylesheet"><style>body { max-width: 840px; margin: 20px auto; font-size: 16px; }</style></head><body>"
tailhtml="</body></html>"
echo "$headhtml" > tmp_doc_head
echo "$tailhtml" > tmp_doc_tail

# main package docs
godoc -url pkg/$pkgurl | tail -n +44 > tmp_doc_body
cat tmp_doc_head tmp_doc_body tmp_doc_tail > $docsfolder$docsprefix$pkgname.html
for pkg in $subpkgs; do
	sed -i "s|/pkg/$pkgurl/$pkg/|./$docsprefix$pkg.html|g" "$docsfolder$docsprefix$pkgname.html"
	sed -i "s|/pkg/golang.org/|https://pkg.go.dev/golang.org/|g" "$docsfolder$docsprefix$pkgname.html"
done

# subpackage docs
for pkg in $subpkgs; do
	godoc -url pkg/$pkgurl/$pkg | tail -n +44 > tmp_doc_body
	cat tmp_doc_head tmp_doc_body tmp_doc_tail > "$docsfolder$docsprefix$pkg.html"
	for pkg in $packages; do
		sed -i "s|/pkg/$pkgurl/$pkg/|./$docsprefix$pkg.html|g" "$docsfolder$docsprefix$pkg.html"
		sed -i "s|/pkg/golang.org/|https://pkg.go.dev/golang.org/|g" "$docsfolder$docsprefix$pkg.html"
	done
done

# clear temp files
rm tmp_doc_head tmp_doc_body tmp_doc_tail
