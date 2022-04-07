// The ecache subpackage defines the GlyphCacheHandler interface used
// within etxt, provides a default cache implementation and exposes a
// few more helper types to assist you if you want to implement your own.
//
// Since glyph rasterization is usually an expensive CPU process, caches are
// a vital part of any real-time text rendering pipeline.
//
// As far as practical advice goes, "how to determine the size of my cache"
// would be the main topic of discussion. Sadly, there's no good rule of
// thumb to say "set your cache size to half its peak memory usage" or
// similar. Cache sizes really depend on your use-case: sometimes you have
// only a couple fonts at a few fixed sizes and you want your cache to fit
// everything. Sometimes you determine your font sizes based on the current
// screen size and can absolutely not pretend to cache all the masks that
// the renderers may generate. The PeakCacheSize function is a good tool
// to assist you, but you will have to figure out your requirements by
// yourself... or just set an arbitrary cache size like 16MB (16*1024*1024
// bytes) and see how far does that get you.
package ecache
