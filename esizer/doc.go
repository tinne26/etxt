// The esizer subpackage defines the Sizer interface used within etxt
// and provides multiple ready-to-use implementations.
//
// The only job of a Sizer is to determine how much space should be taken
// by each glyph. While font files already contain this information,
// using an interface as a middle layer allows etxt users to modify
// spacing manually and achieve specific effects like ignoring kerning,
// adding extra padding between letters or accounting for the extra space
// taken by custom glyph rasterizers.
package esizer
