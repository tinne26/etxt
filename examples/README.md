# etxt examples

This folder contains two subfolders:
- `gtxt` is the basic examples folder. The code uses the generic **etxt** version (`-tags gtxt`), but the techniques showcased have general applicability. This folder contains many simple examples that will help you get started with etxt.
- `ebiten` contains examples on how to use **etxt** with Ebitengine. These tend to be more advanced than `gtxt` examples, but the end results are also more inspiring. If you want to see what etxt is capable of, this is the place.

As long as you have Golang installed (>=go1.18), you can run the examples directly without any previous step. Almost all the programs expect one argument with a path to the font to use[^1]:
```
go run -tags gtxt github.com/tinne26/etxt/examples/gtxt/sizer_expand@latest path/to/your_font.ttf
```

[^1]: If you need a quick font download, you can just pick [Liberation Sans from this link](https://github.com/tinne26/fonts/blob/main/liberation/lbrtsans/LiberationSans-Regular.ttf). It's the sans-serif version of the font embedded for the example on etxt's readme (`examples/ebiten/words`).

For Ebitengine examples, omit the `gtxt` tag:
```
go run github.com/tinne26/etxt/examples/ebiten/colorful@latest path/to/your_font.ttf
```

Alternatively, if you are feeling lazy, you can visit https://tinne26.github.io/etxt-examples/ for some web-based example ports. Or you can check a few `gtxt` results below:

### gtxt/aligns
![](https://raw.githubusercontent.com/tinne26/etxt/v0.0.9-alpha.8/docs/img/gtxt_aligns.png)

### gtxt/quantization
![](https://raw.githubusercontent.com/tinne26/etxt/v0.0.9-alpha.8/docs/img/gtxt_quantization.png)

### gtxt/outline_cheap
![](https://raw.githubusercontent.com/tinne26/etxt/v0.0.9-alpha.8/docs/img/gtxt_outline_cheap.png)

### gtxt/mirror
![](https://raw.githubusercontent.com/tinne26/etxt/v0.0.9-alpha.8/docs/img/gtxt_mirror.png)
