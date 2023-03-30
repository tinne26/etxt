// The cache subpackage defines the [GlyphCacheHandler] interface used
// within etxt and provides a default cache implementation.
//
// Since glyph rasterization is an expensive CPU process, caches are a
// vital part of any real-time text rendering pipeline.
//
// As far as practical advice goes, "how to determine the size of my cache"
// would be the main topic of discussion. Sadly, there's no good rule of
// thumb to say "set your cache size to half its peak memory usage" or
// similar. Cache sizes really depend on your use-case: sometimes you have
// only a couple fonts at a few fixed sizes and you want your cache to fit
// everything. Sometimes you determine your font sizes based on the current
// screen size and can absolutely not pretend to cache all the masks that
// the renderers may generate. The [DefaultCache.PeakSize]() function is
// a good tool to assist you, but you will have to figure out your requirements
// by yourself. Of course, you can also just use etxt.Cache8MiB and see how far
// does that get you.
//
// To give a more concrete size reference, though, let's assume a normal or
// small reading font size, where each glyph mask is around 11x11 on average
// (many glyphs don't have ascenders or descenders). That's about 676 bytes per
// mask on Ebitengine. Then say we will have around 64 different glyphs (there
// may only be 26 letters in english, but we also need to account for uppercase,
// numbers, punctuation, variants with diacritic marks, etc.). We would already
// be around 42KiB of data. If you account for a couple different fonts being 
// used in an app, bigger sizes and maybe variants with italics or bold, you get
// closer to be working with MiBs of data, not KiBs. If you also disable full
// quantization, each glyph mask will need to be rendered for different subpixel
// positions. This can range from anywhere between x2 to x64 memory usage in most
// common scenarios.
//
// The summary would be that anything below 64KiB of cache is almost sure to fall
// short in many scenarios, with a few MiBs of capacity probably being a much
// better ballpark estimate for what many games and applications will end up using
// on their UI screens.
package cache
