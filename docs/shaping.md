# Text shaping
In Arabic, letters are written in different forms based on their position within a word. In Devaganari (Indic), groups of consonants may require ligatures or different glyph forms. Khmer (Combodian script) has diacritic and syllable-modifying marks... These scripts, among many others, are known as [**complex scripts**](https://en.wikipedia.org/wiki/Complex_text_layout), scripts where the shape or positioning of graphemes can vary based on their relation to other graphemes.

In contrast, Latin, Cyrillic, Greek, Hiragana and many others are examples of non-complex scripts.

When it comes to complex scripts, Unicode doesn't include code points for all the possible ligatures, clusters and glyph variations. This means the basic process used for non-complex scripts of mapping Unicode code points to font glyphs is not enough. Instead, a process known as **text shaping** is required to convert an input text to an output sequence of glyphs that takes into account the specific scripts and fonts being used.

This is a complex process that can vary significantly for each script, requiring lots of specific knowledge and individualized handling. HarfBuzz is one of the most mature text shaping libraries in use nowadays. You can read their own definition of text shaping at https://harfbuzz.github.io/what-is-harfbuzz.html.

## etxt support for text shaping
**etxt** doesn't do or offer any tools to do text shaping, but it allows the results of a text shaping process to be used to draw text when passed as slices of [glyph indices](https://pkg.go.dev/golang.org/x/image/font/sfnt#GlyphIndex). All the `*Glyph*` methods in **etxt** have been added with complex script support in mind, and are intended to be used only if you are also doing text shaping.

Sadly, there's a hole in Go's landscape when it comes to text shaping: the most official package for font manipulation in Golang, [**sfnt**](https://pkg.go.dev/golang.org/x/image/font/sfnt) does not expose the GSUB and GPOS font tables required to implement text shaping on your own. This forces Golang programmers to either:
- Fork or reimplement **sfnt** functionality before being able to work on text shaping (or directly contribute to move https://github.com/golang/go/issues/45325 forward).
- Use CGO bindings to bigger libraries like HarfBuzz. See https://pkg.go.dev/github.com/npillmayer/gotype/engine/text/textshaping.
- Reimplement bigger libraries like HarfBuzz in pure Go. See https://github.com/go-text/typesetting.

This is a sad situation because while universal text shaping is a gigantic ~~mess~~ problem and it would be quite insane to attempt to roll your own solution when HarfBuzz already exists, the truth is that in some contexts like indie game development, doing text shaping for a single language (e.g, your own) and a controlled set of fonts would be perfectly reasonable. Instead, right now you are forced to either go big or go home.

To be completely honest, I don't believe in Unicode or [SFNT font formats](https://en.wikipedia.org/wiki/SFNT) at all (they are too big and complex for their own or anyone's good), so I see text shaping as just another layer on top of a broken foundation, but whatever. Hopefully this document gave you some context and helped you understand the relationship between scripts, fonts and Unicode a bit better.
