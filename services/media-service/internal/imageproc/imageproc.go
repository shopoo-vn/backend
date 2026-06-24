package imageproc

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"sync"

	"github.com/disintegration/imaging"
)

// Variant describes one derived size to produce from a source image.
type Variant struct {
	// Name is the size label, e.g. "thumb" / "medium" / "full".
	Name string
	// Width is the target width in pixels. The height is computed to preserve
	// the aspect ratio. A Width of 0 means "keep the original dimensions"
	// (used for the re-encoded full image).
	Width int
}

// Result is the encoded bytes for one variant.
type Result struct {
	Name  string
	Bytes []byte
}

// jpegQuality is the encode quality used for every derived image; re-encoding
// the original as JPEG both normalises the format and compresses it.
const jpegQuality = 85

// Processor decodes a source image once and resizes it into several variants
// using a bounded worker pool so a burst of large uploads cannot exhaust memory
// by spawning an unbounded number of goroutines.
type Processor struct {
	workers int
}

// NewProcessor returns a Processor whose fan-out is capped at workers. A value
// below 1 is clamped to 1.
func NewProcessor(workers int) *Processor {
	if workers < 1 {
		workers = 1
	}
	return &Processor{workers: workers}
}

// Process decodes src and produces every requested variant as JPEG bytes. Work
// is fanned out across at most p.workers goroutines (a counting semaphore); the
// passed context cancels in-flight and pending work. The returned slice
// preserves the order of variants.
func (p *Processor) Process(ctx context.Context, src []byte, variants []Variant) ([]Result, error) {
	img, err := imaging.Decode(bytes.NewReader(src), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	results := make([]Result, len(variants))
	errs := make([]error, len(variants))

	// Counting semaphore bounds concurrency to p.workers; every goroutine is
	// guaranteed to exit because each acquires and releases exactly one slot.
	sem := make(chan struct{}, p.workers)
	var wg sync.WaitGroup

	for i, v := range variants {
		// Honour cancellation before scheduling more work.
		if err := ctx.Err(); err != nil {
			errs[i] = err
			continue
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			errs[i] = ctx.Err()
			continue
		}

		wg.Add(1)
		go func(idx int, variant Variant) {
			defer wg.Done()
			defer func() { <-sem }()

			b, encErr := encodeVariant(img, variant)
			if encErr != nil {
				errs[idx] = encErr
				return
			}
			results[idx] = Result{Name: variant.Name, Bytes: b}
		}(i, v)
	}

	wg.Wait()

	for _, e := range errs {
		if e != nil {
			return nil, e
		}
	}
	return results, nil
}

// encodeVariant resizes (if needed) and JPEG-encodes a single variant.
func encodeVariant(img image.Image, v Variant) ([]byte, error) {
	out := img
	if v.Width > 0 && img.Bounds().Dx() > v.Width {
		// Height 0 preserves the aspect ratio. Lanczos gives good downscale quality.
		out = imaging.Resize(img, v.Width, 0, imaging.Lanczos)
	}
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, out, imaging.JPEG, imaging.JPEGQuality(jpegQuality)); err != nil {
		return nil, fmt.Errorf("encode %s: %w", v.Name, err)
	}
	return buf.Bytes(), nil
}
