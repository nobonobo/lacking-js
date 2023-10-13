//go:build dfp

package app

import (
	"encoding/binary"
	"log"
	"math"
	"sync"
	"syscall/js"
	"time"

	"github.com/mokiat/gomath/dprec"
	"github.com/mokiat/lacking/app"
)

// NOTE: Chrome does not follow the specification and the Gamepad object
// reference cannot be stored and reused. It contains a snapshot of some
// state which does not get updated. This makes using the connect and disconnect
// event handlers pointless.
//
// Shame...

func newGamepad(index int) *Gamepad {
	g := &Gamepad{
		index: index,

		isDirty:     true,
		isConnected: false,
		isSupported: false,

		deadzoneStick:   0.0,
		deadzoneTrigger: 0.0,
	}
	g.connect = sync.OnceFunc(g.initialize)
	return g
}

type Gamepad struct {
	index int

	connect func()
	device  js.Value
	pulse   chan float64

	isDirty     bool
	isConnected bool
	isSupported bool

	deadzoneStick   float64
	deadzoneTrigger float64

	leftStickX        float64
	leftStickY        float64
	leftStickButton   bool
	rightStickX       float64
	rightStickY       float64
	rightStickButton  bool
	leftBumperButton  bool
	leftTrigger       float64
	rightBumperButton bool
	rightTrigger      float64
	dpadLeftButton    bool
	dpadRightButton   bool
	dpadUpButton      bool
	dpadDownButton    bool
	actionLeftButton  bool
	actionRightButton bool
	actionUpButton    bool
	actionDownButton  bool
	forwardButton     bool
	backButton        bool
}

var _ app.Gamepad = (*Gamepad)(nil)

func (g *Gamepad) Connected() bool {
	g.refresh()
	return g.isConnected
}

func (g *Gamepad) Supported() bool {
	g.refresh()
	return g.isSupported
}

func (g *Gamepad) StickDeadzone() float64 {
	return g.deadzoneStick
}

func (g *Gamepad) SetStickDeadzone(deadzone float64) {
	g.deadzoneStick = deadzone
}

func (g *Gamepad) TriggerDeadzone() float64 {
	return g.deadzoneTrigger
}

func (g *Gamepad) SetTriggerDeadzone(deadzone float64) {
	g.deadzoneTrigger = deadzone
}

func (g *Gamepad) LeftStickX() float64 {
	g.refresh()
	return deadzoneValue(g.leftStickX, g.deadzoneStick)
}

func (g *Gamepad) LeftStickY() float64 {
	g.refresh()
	return deadzoneValue(g.leftStickY, g.deadzoneStick)
}

func (g *Gamepad) LeftStickButton() bool {
	g.refresh()
	return g.leftStickButton
}

func (g *Gamepad) RightStickX() float64 {
	g.refresh()
	return deadzoneValue(g.rightStickX, g.deadzoneStick)
}

func (g *Gamepad) RightStickY() float64 {
	g.refresh()
	return deadzoneValue(g.rightStickY, g.deadzoneStick)
}

func (g *Gamepad) RightStickButton() bool {
	g.refresh()
	return g.rightStickButton
}

func (g *Gamepad) LeftTrigger() float64 {
	g.refresh()
	return deadzoneValue(g.leftTrigger, g.deadzoneTrigger)
}

func (g *Gamepad) RightTrigger() float64 {
	g.refresh()
	return deadzoneValue(g.rightTrigger, g.deadzoneTrigger)
}

func (g *Gamepad) LeftBumper() bool {
	g.refresh()
	return g.leftBumperButton
}

func (g *Gamepad) RightBumper() bool {
	g.refresh()
	return g.rightBumperButton
}

func (g *Gamepad) DpadUpButton() bool {
	g.refresh()
	return g.dpadUpButton
}

func (g *Gamepad) DpadDownButton() bool {
	g.refresh()
	return g.dpadDownButton
}

func (g *Gamepad) DpadLeftButton() bool {
	g.refresh()
	return g.dpadLeftButton
}

func (g *Gamepad) DpadRightButton() bool {
	g.refresh()
	return g.dpadRightButton
}

func (g *Gamepad) ActionUpButton() bool {
	g.refresh()
	return g.actionUpButton
}

func (g *Gamepad) ActionDownButton() bool {
	g.refresh()
	return g.actionDownButton
}

func (g *Gamepad) ActionLeftButton() bool {
	g.refresh()
	return g.actionLeftButton
}

func (g *Gamepad) ActionRightButton() bool {
	g.refresh()
	return g.actionRightButton
}

func (g *Gamepad) ForwardButton() bool {
	g.refresh()
	return g.forwardButton
}

func (g *Gamepad) BackButton() bool {
	g.refresh()
	return g.backButton
}

func (g *Gamepad) Pulse(intensity float64, duration time.Duration) {
	g.pulse <- intensity
}

func (g *Gamepad) markDirty() {
	g.isDirty = true
}

func (g *Gamepad) hidDevice() js.Value {
	if g.device.IsUndefined() || g.device.IsNull() {
		g.connect()
	}
	return g.device
}

func (g *Gamepad) refresh() {
	if !g.isDirty {
		return
	}
	device := g.hidDevice()
	g.isDirty = false
	g.isConnected = !device.IsUndefined() && !device.IsNull()
	if g.isConnected {
		g.isSupported = true
	} else {
		g.isSupported = false
	}
}

func deadzoneValue(value, deadzone float64) float64 {
	if math.Signbit(value) {
		// negative
		value = dprec.Max(-value, deadzone)
		value = value - deadzone
		return -value / (1.0 - deadzone)
	} else {
		// positive
		value = dprec.Max(value, deadzone)
		value = value - deadzone
		return value / (1.0 - deadzone)
	}
}

const (
	vendorId  = 0x046d
	productId = 0xc298
)

var (
	alert = js.Global().Get("alert")
	hid   = js.Global().Get("navigator").Get("hid")
)

func (g *Gamepad) rxInputReport(this js.Value, args []js.Value) any {
	ev := args[0]
	id := ev.Get("reportId").Int()
	b := JS2Bytes(ev.Get("data"))
	switch id {
	case 0:
		head := binary.LittleEndian.Uint16(b[0:2])
		g.leftStickX = float64(int16(head&0x3fff)-0x2000) / 0x2000
		g.actionLeftButton = head&0x8000 != 0
		g.actionDownButton = head&0x4000 != 0
		g.actionUpButton = b[2]&0x02 != 0
		g.actionRightButton = b[2]&0x01 != 0
		g.leftTrigger = dprec.Clamp(float64(240-int16(b[6]))/240, float64(0), float64(1))
		g.rightTrigger = dprec.Clamp(float64(240-int16(b[5]))/240, float64(0), float64(1))
		hat := b[3] >> 4
		g.dpadUpButton = hat == 0 || hat == 1 || hat == 7
		g.dpadRightButton = hat == 1 || hat == 2 || hat == 3
		g.dpadDownButton = hat == 3 || hat == 4 || hat == 5
		g.dpadLeftButton = hat == 5 || hat == 6 || hat == 7
		g.forwardButton = b[3]&0x08 != 0
		g.backButton = b[3]&0x04 != 0
		//log.Printf("rx: %x/%x, s=%4.1f", id, b, g.leftStickX)
	default:
		log.Printf("rx: %x/%x, s=%4.1f", id, b, g.leftStickX)
	}
	return nil
}

func (g *Gamepad) initialize() {
	log.Println("connecting...")
	go func() {
		devices, err := Await(hid.Call("getDevices"))
		if err != nil {
			alert.Invoke(err.Error())
			g.device = js.Null()
			return
		}
		log.Println(devices)
		fn := js.FuncOf(func(this js.Value, args []js.Value) any {
			return args[0].Get("vendorId").Int() == vendorId && args[0].Get("productId").Int() == productId
		})
		dev := devices.Call("find", fn)
		fn.Release()
		if dev.IsNull() || dev.IsUndefined() {
			alert.Invoke(err.Error())
			g.device = js.Null()
			return
		}
		if !dev.Get("opened").Bool() {
			if _, err := Await(dev.Call("open")); err != nil {
				alert.Invoke(err.Error())
				return
			}
		}
		dev.Call("addEventListener", "inputreport", js.FuncOf(g.rxInputReport))
		Await(dev.Call("sendReport", 0x00, Bytes2JS([]byte{0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})))
		Await(dev.Call("sendReport", 0x00, Bytes2JS([]byte{0xfe, 0x0d, 0x0c, 0x0c, 0x80, 0x00, 0x00})))
		Await(dev.Call("sendReport", 0x00, Bytes2JS([]byte{0x11, 0x08, 0x80, 0x80, 0x00, 0x00, 0x00})))
		Await(dev.Call("sendReport", 0x00, Bytes2JS([]byte{0x21, 0x0c, 0x01, 0x00, 0x01, 0x00, 0x01})))
		//Await(dev.Call("sendReport", 0x00, Bytes2JS([]byte{0x13, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})))
		g.pulse = make(chan float64, 16)
		go func() {
			for v := range g.pulse {
				v := byte(uint(dprec.Clamp(v*128+128, float64(0), float64(255))))
				Await(dev.Call("sendReport", 0x00, Bytes2JS([]byte{0x11, 0x08, v, 0x80, 0x00, 0x00, 0x00})))
			}
		}()
		g.device = dev
		log.Println("connect:", g.device.Get("productName"))
	}()
}

func GamepadConnect() {
	devices, err := Await(hid.Call("getDevices"))
	if err != nil {
		alert.Invoke(err.Error())
		return
	}
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		return args[0].Get("vendorId").Int() == vendorId && args[0].Get("productId").Int() == productId
	})
	dev := devices.Call("find", fn)
	fn.Release()
	if dev.IsNull() || dev.IsUndefined() {
		devices, err := Await(hid.Call("requestDevice", map[string]any{
			"filters": []any{map[string]any{"productId": productId, "vendorId": vendorId}}}))
		if err != nil {
			alert.Invoke(err.Error())
			return
		}
		if devices.Length() > 0 {
			dev = devices.Index(0)
		}
	}
	if dev.IsNull() || dev.IsUndefined() {
		alert.Invoke("No device found")
		return
	}
	log.Println(dev.Get("productName").String())
}
