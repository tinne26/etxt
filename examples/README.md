# etxt examples
This folder contains two subfolders:
- `gtxt` is the main examples folder. The code uses the generic **etxt** version (`-tags gtxt`), but the techniques showcased have general applicability. This folder contains many simple examples that will help you get started. Almost all the `gtxt` examples expect one argument with a path to the font to use.
- `ebiten` contains examples on how to use **etxt** with Ebiten. The examples are mostly advanced, including animation effects, shaders and similar. I recommend starting with the examples in `gtxt` instead.

As long as you have Golang installed, you can run the examples directly without any previous step. E.g. (replace `path/to/your_font.ttf` with an actual path):
```
go run -tags gtxt github.com/tinne26/etxt/examples/gtxt/sizer_expand@latest path/to/your_font.ttf
```

For Ebiten examples, don't use the `gtxt` tag:
```
go run github.com/tinne26/etxt/examples/ebiten/typewriter@latest path/to/your_font.ttf
```

Below you can see the results of a few examples:

### gtxt/aligns
![](https://github.com/tinne26/etxt/blob/main/docs/img/gtxt_aligns.png?raw=true)

### gtxt/quantization
![](https://github.com/tinne26/etxt/blob/main/docs/img/gtxt_quantization.png?raw=true)

### gtxt/outline
![](https://github.com/tinne26/etxt/blob/main/docs/img/gtxt_outline.png?raw=true)

### gtxt/mirror
![](https://github.com/tinne26/etxt/blob/main/docs/img/gtxt_mirror.png?raw=true)
