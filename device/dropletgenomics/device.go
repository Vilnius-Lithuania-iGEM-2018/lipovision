//dropletgenomics package defines the DropletGenomics company's microfluidincs device
package dropletgenomics

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"time"
)

//CreateDropletGenomicsDevice returns a device configured with given configuration
func CreateDropletGenomicsDevice() Device {
	return Device{}
}

//Device is DropletGenomics' rendition of microfluidics devices
type Device struct {
	IPAddress         string
	HTTPPort          int
	PumpDataPort      int
	RecordingDataPort int
	PumpExperiment    int
	pumps             []Pump
	camera            Camera
}

//Stream launches async stream decoding of ctx lifetime
func (device Device) Stream(ctx context.Context) <-chan Frame {
	const (
		streamEndpoint string = "http://192.168.1.100:8765/video_feed"
		frameRate      int64  = 30
	)

	stream := make(chan Frame, 10)
	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			default:
				response, err := http.Get(streamEndpoint)
				if err != nil {
					fmt.Printf("Failed to connect to stream")
					continue
				}
				byteStream := response.Body

				var decodeError error
				for decodeError == nil {
					var img image.Image

					buffer := make([]byte, 36, 36)
					byteStream.Read(buffer)

					img, decodeError = png.Decode(byteStream)
					frameLifetime, cancel := context.WithTimeout(ctx, time.Second/(time.Duration)(frameRate))
					stream <- Frame{frame: img, ctx: frameLifetime, cancel: cancel}
				}
				byteStream.Close()
			}
		}
	}()
	return stream
}

// Available determines whether connection to
// DropletGenomics device is established
func (device *Device) Available() bool {
	url := setupDeviceURL(device)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return false
	}
	return true
}

//Camera returns the device's camera data
func (device Device) Camera() *Camera {
	return &device.camera
}

//Pump returns device's pump by it's id
func (device Device) Pump(index int) *Pump {
	return &device.pumps[index]
}

//RefreshAll launches refresh on all pumps
func (device Device) RefreshAll() {
	for _, pump := range device.pumps {
		pump.Invoke(PumpRefresh, nil)
	}
}

//EstablishPumpsWithID creates a pump with ID
func (device Device) EstablishPumpsWithID() {
	device.pumps = make([]Pump, device.PumpExperiment, device.PumpExperiment)
	for i := 0; i < device.PumpExperiment; i++ {
		device.pumps[i] = NewPump(i)
	}
}
