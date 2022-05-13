# Rasterizing outlines
This document aims to give a general overview of the rasterization algorithms used by [`vector.Rasterizer`](https://pkg.go.dev/golang.org/x/image/vector) and [`emask.EdgeMarker`](https://pkg.go.dev/github.com/tinne26/etxt/emask#EdgeMarkerRasterizer).

While this package focuses on glyph rendering, these algorithms are suitable for general 2D vector graphics. That said, they are CPU-based processes best suited for small shapes; in the case of big shapes, GPU algorithms based on triangulation and [spline curve rendering](https://developer.nvidia.com/gpugems/gpugems3/part-iv-image-effects/chapter-25-rendering-vector-art-gpu) may be a better choice.

## The problem
Given a vectorial outline, we want to rasterize it (convert the outline to a raster image, which is a grid of square pixels). Here's an example of outlines and raster:

![](https://github.com/tinne26/etxt/blob/main/docs/img/outline_vs_raster.png?raw=true)

We will call the resulting raster image a *mask*, as it only contains information about the opacity of each pixel, not about their colors. Colors can be chosen and applied later, at a separate step.

Outlines are defined using two primitive operations:
- Moving the "pen position" to a specific coordinate.
- Adding a straight segment from the current "pen position" to a new position.

While quadratic and cubic Bézier curves are also allowed in `vector.Rasterizer` and `emask.EdgeMarker`, those aren't primitives; curves are internally converted to multiple straight lines before processing.

Notice also that while we often talk about lines and pen positions and outlines, it's best that you think in terms of "boundaries" and "delimiting segments". Boundaries have no width, what matters is the areas they delimit.

## Allowing outlines with holes
Each outline can have one or more contours, which are closed shapes delimiting an outer or inner region of the outline. For example:
- The outline of a "1" generally has only one outer contour.
- The outline of a "6" generally has two contours, one being an inner contour (the hole), and the other being an outer contour.
- The outline of a "8" generally has three contours. In this case, we have two inner contours and one outer contour.

In order to make "holes" work, the standard procedure is to consider the contour direction:
- If two overlapping contours are defined in opposite directions (clockwise and counter-clockwise, or viceversa), we will make their overlapping area be unfilled.
- If contours share the same orientation, overlapping areas will stay filled.

## How to solve the problem
Now that we are situated, we can finally ask ourselves **how to go from outlines to a mask**.

Here comes a first idea: determine the bounds of the outline, allocate an image big enough to contain it, and then for each pixel in the image determine if it's inside or outside the outlines.

This is a good general description of what we want to achieve, but there are a few big problems:
- How do we determine if a point is inside or outside an outline?
- What if a pixel is not fully inside/outside an outline, but only partially?
- If outlines can have holes... inside an outline doesn't necessarily mean inside the filled region. In fact, if outlines intersect, maybe a contour can be both inner and outer at the same time...

There are multiple answers to each of these questions:
- We could triangulate the outline first and then check points against our set of triangles to determine if they overlap (or iterate the areas of the triangles instead to be more efficient).
- If a pixel is 30% covered by the outline, we can set its opacity to 30%. Keep opacity proportional to "how much the pixel is filled".
- We could fill each contour one by one, count how many outlines are affecting a pixel and use an even-odd rule or similar to fill or unfill based on how many times we have touched each pixel.

...but while triangulation is a suitable method for big shapes to be processed on the GPU, on CPU we will use another approach: marking the outline boundaries.

## Marking outline boundaries
Let's say we have a glyph like this:

![](https://github.com/tinne26/etxt/blob/main/docs/img/glyph_filled.png?raw=true)

Now, starting from the left side, we start going towards the right and each time we cross an outline boundary, we mark it. The result would be something like this:

![](https://github.com/tinne26/etxt/blob/main/docs/img/glyph_edges.png?raw=true)

Well, that's the core idea that will help us solve our problem. Each time we issue a `LineTo` command to define a boundary segment for a contour, we will follow the line, see which pixels it crosses, and somehow store that information.

This data can be stored internally in a regular array or buffer (dense representation), or using a sparse(r) representation. For the sake of simplicity we will be using a buffer, but both are used in practice and offer different trade-offs. You could even use a hybrid approach based on the final size of the mask.

We are making progress now, but there are still a few loose ends...

First, to account for clockwise and counter-clockwise directions and make "holes" possible, we will make positive y changes (upward lines) set positive crossing values, and negative y changes (downward lines) set negative crossing values.

If we use different colors for positive and negative changes, the result would now look like this:

![](https://github.com/tinne26/etxt/blob/main/docs/img/glyph_sign.png?raw=true)

The important part is that different directions result in values of opposite sign (e.g.: you could make "up" be negative and "down" positive instead).

Now that we have an outline closer to what we want, it's time to dive into the final details.

If you look at the previous outlines, you will see that the fully horizontal boundaries are not marked. This is key: since boundaries do not have any thickness and we don't have an infinitesimal grid (but rather a chunky pixel grid), what we have to mark are not the boundaries themselves, but *the regions through which we cross them*.

To do this, we need to choose a consistent direction to annotate all "crossings". To play nicely with computer memory layouts, we mark boundaries when they cross pixels *vertically*. Then, at a later step, we can traverse our buffer from left to right (horizontally) and fill the outline using accumulators. It's ok if you don't fully understand this right now, just remember that *we adjust the pixel opacities when boundaries cross them vertically*.

To illustrate the concept more practically, let's imagine we have a mask with a single pixel:
- If we create an outline that starts at `(0, 0)` and goes to `(0, 1)`, `(1, 1)`, `(0, 1)` and finally back to the start (creating a pixel-sized square), since the first `(0, 0) -> (0, 1)` boundary crosses the pixel vertically in full and does so from its horizontal start, we would set the opacity of that pixel to 1. Horizontal boundaries like `(0, 1) -> (1, 1)`, instead, *don't change opacity*. The other vertical boundary going back to `y = 0` would decrease the opacity, but it would do so in the next pixel to the right, not on the current one.
- If we made a narrower rectangle starting at `x = 0.5`, the vertical area would still be fully traversed, but now since the boundary starts at half the pixel, we would only set its value to half the opacity. But this extremely important! Since we have fully traversed the vertical area, the opacity of the next pixels (going to the right) would have to become 100% later anyway! As an **invariant**, *each crossing that we mark has to add opacity proportional to how much we moved vertically*. Since the current pixel can't take 100% opacity, only 50%, the remaining 50% would go to the next one. This may sound strange at the beginning, but if you apply these rules consistently you can use the pixel opacities as accumulator values and determine the opacity of each pixel by scanning them from left to right (well, technically the magnitude; the opacity sign may vary).

I'd explain more, but at this point you have enough context and jumping directly [into the code](https://github.com/tinne26/etxt/blob/main/emask/edge_marker.go) may be the best next step.

## Limitations
This algorithm works decently in general, but notice that what happens inside a pixel can only be balanced, not distinguished. For example, if we define a 1x1 square in the middle of four pixels, we will get 25% opacity from each pixel. That's the best we can do, ok... but if you repeat the process 4 times, the 4 pixels will all get to 100% opacity. This wouldn't happen in a continuous space, but since pixels can't tell where lines start or end within them (discrete space), they can't tell it's always the same area being covered and the result "overflows".

Floating point precision can also be an issue when dealing with big shapes, unaligned and angled boundaries, etc.

## Further references
Curve segmentation and optimizations haven't been discussed, but are mentioned here and there through the code.

You will also be interested in Raph Levien's [blogpost](https://medium.com/@raphlinus/inside-the-fastest-font-renderer-in-the-world-75ae5270c445) explaining the [font-rs](https://github.com/raphlinus/font-rs) font renderer, which in turn serves as a base for Golang's `vector.Rasterizer` [implementation](https://cs.opensource.google/go/x/image/+/70e8d0d3:vector/raster_floating.go;l=31).
