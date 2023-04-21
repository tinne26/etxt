#!/bin/bash

# Generates docs/reference_pkg.html files for each package, slightly nicer
# than what godoc generates by default. Requires having godoc installed.

# You can open the results after running the script with:
# >> xdg-open docs/reference_etxt.html

pkgname="etxt"
pkgurl="github.com/tinne26/$pkgname"
subpkgs="mask fract sizer cache"
docsfolder="docs/"
docsprefix="reference_"

# download css and js files if necessary. they will be placed on docs/
# and you may remove them at any time in order to re-fetch them.
cssurl="https://raw.githubusercontent.com/golang/tools/master/godoc/static/style.css"
jsurl="https://raw.githubusercontent.com/golang/tools/master/godoc/static/godocs.js"
jqueryurl="https://raw.githubusercontent.com/golang/tools/master/godoc/static/jquery.js"
cssfile=reference_style.css
jsfile=reference_js.js
jqueryfile=reference_jquery.js

if [ ! -f "./docs/$cssfile" ]; then
	echo "downloading godoc css..."
	curl -sS $cssurl --output ./docs/$cssfile
fi
if [ ! -f "./docs/$jsfile" ]; then
	echo "downloading godoc js..."
	curl -sS $jsurl --output ./docs/$jsfile
fi
if [ ! -f "./docs/$jqueryfile" ]; then 
	echo "downloading jquery..."
	curl -sS $jqueryurl --output ./docs/$jqueryfile
fi

headhtml="<!DOCTYPE html><html><head><title>$pkgurl/godoc</title><link href=\"$cssfile\" rel=\"stylesheet\"><style>body { max-width: 900px; margin: 20px auto; font-size: 16px; text-align: left; }</style><script src=\"$jqueryfile\"></script><script src=\"$jsfile\"></script></head><body>"
tailhtml="</body></html>"
echo "$headhtml" > tmp_doc_head
echo "$tailhtml" > tmp_doc_tail

# main package docs
echo "generating docs for $pkgurl..."
godoc -url pkg/$pkgurl | tail -n +44 > tmp_doc_body
cat tmp_doc_head tmp_doc_body tmp_doc_tail > $docsfolder$docsprefix$pkgname.html
for pkg in $subpkgs; do
	sed -i "s|/pkg/$pkgurl/$pkg/|./$docsprefix$pkg.html|g" "$docsfolder$docsprefix$pkgname.html"
	sed -i "s|/pkg/golang.org/|https://pkg.go.dev/golang.org/|g" "$docsfolder$docsprefix$pkgname.html"
done

# subpackage docs
for pkg in $subpkgs; do
	echo "generating docs for subpackage $pkg..."
	godoc -url pkg/$pkgurl/$pkg | tail -n +44 > tmp_doc_body
	cat tmp_doc_head tmp_doc_body tmp_doc_tail > "$docsfolder$docsprefix$pkg.html"
	for pkg in $packages; do
		sed -i "s|/pkg/$pkgurl/$pkg/|./$docsprefix$pkg.html|g" "$docsfolder$docsprefix$pkg.html"
		sed -i "s|/pkg/golang.org/|https://pkg.go.dev/golang.org/|g" "$docsfolder$docsprefix$pkg.html"
	done
done

# clear temp files
rm tmp_doc_head tmp_doc_body tmp_doc_tail
