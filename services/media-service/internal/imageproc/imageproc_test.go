package imageproc

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/disintegration/imaging"
)

// makeJPEG returns an in-memory JPEG of the given dimensions for test input.
func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.JPEG); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

func decodeDims(t *testing.T, b []byte) (int, int) {
	t.Helper()
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func TestProcess_ProducesAllVariants(t *testing.T) {
	src := makeJPEG(t, 2000, 1000)
	p := NewProcessor(2)

	variants := []Variant{
		{Name: "thumb", Width: 256},
		{Name: "medium", Width: 1024},
		{Name: "full", Width: 0},
	}

	results, err := p.Process(context.Background(), src, variants)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(results) != len(variants) {
		t.Fatalf("got %d results, want %d", len(results), len(variants))
	}

	byName := map[string][]byte{}
	for i, res := range results {
		if res.Name != variants[i].Name {
			t.Fatalf("result %d name = %q, want %q (order not preserved)", i, res.Name, variants[i].Name)
		}
		if len(res.Bytes) == 0 {
			t.Fatalf("result %q has no bytes", res.Name)
		}
		byName[res.Name] = res.Bytes
	}

	if w, _ := decodeDims(t, byName["thumb"]); w != 256 {
		t.Errorf("thumb width = %d, want 256", w)
	}
	if w, _ := decodeDims(t, byName["medium"]); w != 1024 {
		t.Errorf("medium width = %d, want 1024", w)
	}
	if w, _ := decodeDims(t, byName["full"]); w != 2000 {
		t.Errorf("full width = %d, want 2000 (original preserved)", w)
	}
}

func TestProcess_SmallSourceNotUpscaled(t *testing.T) {
	src := makeJPEG(t, 100, 50)
	p := NewProcessor(4)

	results, err := p.Process(context.Background(), src, []Variant{{Name: "thumb", Width: 256}})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if w, _ := decodeDims(t, results[0].Bytes); w != 100 {
		t.Errorf("thumb width = %d, want 100 (no upscaling)", w)
	}
}

func TestProcess_InvalidImage(t *testing.T) {
	p := NewProcessor(1)
	_, err := p.Process(context.Background(), []byte("not an image"), []Variant{{Name: "full"}})
	if err == nil {
		t.Fatal("expected error decoding invalid image, got nil")
	}
}

func TestProcess_CanceledContext(t *testing.T) {
	src := makeJPEG(t, 800, 600)
	p := NewProcessor(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.Process(ctx, src, []Variant{{Name: "thumb", Width: 256}, {Name: "full"}})
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}
