package device

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"
)

type testImage struct {
	underlying int
}

func (t testImage) Get() int {
	return t.underlying
}

func (t testImage) ColorModel() color.Model {
	return nil
}

func (t testImage) At(x, y int) color.Color {
	return nil
}

func (t testImage) Bounds() image.Rectangle {
	return image.Rectangle{}
}

type testFrame struct {
	frame  testImage
	ctx    context.Context
	cancel context.CancelFunc
}

func (f testFrame) Skip() <-chan struct{} {
	return f.ctx.Done()
}

func (f testFrame) Frame() testImage {
	defer f.cancel()
	return f.frame
}

type intProducingDevice struct {
	itemSleepTime time.Duration
}

func (ipd intProducingDevice) Stream(ctx context.Context) <-chan testFrame {
	stream := make(chan testFrame, 5)
	go func() {
		for i := 0; i < 20; i++ {
			// Old frame must be cancelled just before the new frame is dispatched
			frameCtx, cancel := context.WithCancel(ctx)
			stream <- testFrame{frame: testImage{underlying: i + 1}, ctx: frameCtx, cancel: cancel}
			time.Sleep(ipd.itemSleepTime)
			cancel()
		}
	}()
	return stream
}

func (intProducingDevice) Available() bool {
	return true
}

// Must return frames in order and manipulated
func TestGettingFrames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	device := intProducingDevice{itemSleepTime: 5 * time.Millisecond}

	stream := device.Stream(ctx)

	// Imitates a processor
	go func() {
		streamIndex := 0
		for item := range stream {
			select {
			case <-item.Skip():
				t.Error("Frame skipped")
				continue
			default:
			}

			if item.Frame().Get() != streamIndex+1 {
				t.Errorf("Process failed manipulation with: %d", streamIndex+1)
			}
			streamIndex = streamIndex + 1
		}
	}()

	time.Sleep(300 * time.Millisecond)

	select {
	case <-ctx.Done():
	default:
		t.Error("Context did not timeout")
	}

	cancel()
}

// Must skip frames
func TestSkippingFrames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	device := intProducingDevice{itemSleepTime: 5 * time.Millisecond}

	stream := device.Stream(ctx)

	// Imitates a processor
	go func() {
		for item := range stream {

			// Say this is taking too long to process
			time.Sleep(10 * time.Millisecond)

			select {
			case <-item.Skip():
				continue
			default:
				t.Error("Did not skip frame")
			}

			// Should never get here
		}
	}()

	time.Sleep(time.Second / 2)

	select {
	case <-ctx.Done():
	default:
		t.Error("Context did not timeout")
	}

	cancel()
}
