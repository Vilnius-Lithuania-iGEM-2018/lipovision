package main

import (
	"context"
	"image"

	"github.com/Vilnius-Lithuania-iGEM-2018/lipovision/device"
	"github.com/Vilnius-Lithuania-iGEM-2018/lipovision/device/dropletgenomics"
	"github.com/Vilnius-Lithuania-iGEM-2018/lipovision/device/video"
	"github.com/Vilnius-Lithuania-iGEM-2018/lipovision/gui"
	"github.com/Vilnius-Lithuania-iGEM-2018/lipovision/processor"
	"github.com/gotk3/gotk3/gtk"
	log "github.com/sirupsen/logrus"
)

var (
	mainCtx      context.Context
	mainCancel   context.CancelFunc
	activeDevice device.Device
	deviceSet    bool
)

var (
	illuminationValue float64
	exposureValue     float64
)

func chooseFileCreateDevice(win *gtk.Window) device.Device {
	chooser, err := gtk.FileChooserDialogNewWith1Button(
		"Select video file", win, gtk.FILE_CHOOSER_ACTION_OPEN,
		"Open", gtk.RESPONSE_ACCEPT)
	if err != nil {
		log.Fatal("File chooser failed: ", err)
	}
	defer chooser.Destroy()

	filter, _ := gtk.FileFilterNew()
	filter.AddPattern("*.mp4")
	filter.SetName(".mp4")
	chooser.AddFilter(filter)

	resp := chooser.Run()
	log.Info(resp)

	videoFile := chooser.GetFilename()
	return video.Create(videoFile, 24)
}

func main() {
	mainCtx, mainCancel = context.WithCancel(context.Background())
	defer mainCancel()

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window")
	}
	win.SetTitle("LipoVision")
	win.Connect("destroy", func() {
		mainCancel()
		gtk.MainQuit()
	})
	win.SetDefaultSize(900, 600)

	content, err := gui.NewMainControl()
	if err != nil {
		panic(err)
	}
	win.Add(content.Root())
	win.ShowAll()

	registerDeviceChange(content, win)

	gtk.Main()
}

func registerEventHandling(content *gui.MainControl, win *gtk.Window) {
	content.Camera.AutoAdjButton.Connect("clicked", func() {
		activeDevice.Camera().Invoke(device.CameraAutoAdjust, 0)
	})
	content.Camera.IlluminationScale.Connect("format-value", func(scale *gtk.Scale) {
		value := scale.GetValue()
		if illuminationValue != value {
			activeDevice.Camera().Invoke(device.CameraSetIllumination, value)
			illuminationValue = value
		}
	})
	content.Camera.ExposureScale.Connect("format-value", func(scale *gtk.Scale) {
		value := scale.GetValue()
		if exposureValue != value {
			activeDevice.Camera().Invoke(device.CameraSetExposure, value)
			exposureValue = value
		}
	})
}

func registerDeviceChange(content *gui.MainControl, win *gtk.Window) {
	content.StreamControl.ComboBox.Connect("changed", func(combo *gtk.ComboBoxText) {
		mainCancel()
		mainCtx, mainCancel = context.WithCancel(context.Background())
		selection := combo.GetActiveText()
		switch selection {
		case "Video file...":
			activeDevice = chooseFileCreateDevice(win)
		case "DropletGenomics":
			activeDevice = dropletgenomics.Create(4)
		default:
			errDialog := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL,
				gtk.MESSAGE_ERROR, gtk.BUTTONS_OK,
				"Chosen device %s, does not exist", selection)
			errDialog.Run()
		}

		frameHandlers := make(map[string]func(image.Image))
		frameHandlers[processor.StreamOriginal] = func(frame image.Image) {
			if err := content.StreamControl.ShowFrame(frame); err != nil {
				log.Error("Failed to show frame on main window: ", err)
			}
		}

		stream := activeDevice.Stream(mainCtx)
		processor.NewFrameProcessor().Launch(stream, frameHandlers)

		if !deviceSet {
			registerEventHandling(content, win)
		}
	})
}
