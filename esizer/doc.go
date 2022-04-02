// The esizer subpackage contains the definition of the Sizer interface
// used within etxt, along with multiple ready-to-use implementations.
//
// The only job of a Sizer is to determine how much space should be taken
// by each glyph. While this is already indicated by the font data, providing
// an interface that can be used with etxt Renderers allows us to modify
// spacing to achieve some specific effects like ignoring kerning, adding
// extra padding between letters or account for the extra space taken by
// custom glyph mask rasterizers.
package esizer
