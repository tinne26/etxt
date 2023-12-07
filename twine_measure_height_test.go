package etxt

import "testing"

func TestTwineHeightSizerParity(t *testing.T) {
	if testFontA == nil { t.SkipNow() }

	renderer := NewRenderer()
	renderer.SetFont(testFontA)
	renderer.Utils().SetCache8MiB()

	sizer := getTwineHeightSizer()
	defer releaseTwineHeightSizer(sizer)
	var twine Twine

	tests := []string{
		"the only purpose of my life is being measured",
		"hello\ngoodbye",
		"hateful tricky cases\n",
		"\n",
		" \n",
		"\n ",
		" \n\n ",
		"please let\n\nme go",
		"who let\n \nthe dogs out?!",
		"\nwho?!\nwho?!\nwho?! who?!\n",
	}
	
	for i, test := range tests {
		twine.Reset()
		twine.Add(test)
		textHeight  := renderer.Measure(test).Height()
		sizer.Initialize(renderer, twine)
		twineHeight := sizer.Measure(renderer, nil)

		if textHeight != twineHeight {
			textf, twinef := textHeight.ToFloat64(), twineHeight.ToFloat64()
			t.Fatalf("twine measure height parity #%d, expected %f, got %f", i, textf, twinef)
		}
	}
}
