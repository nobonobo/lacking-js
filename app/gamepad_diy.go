//go:build diy

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

const ForceRate = 0.15

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
	device2 js.Value
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
	g.pulse <- dprec.Clamp(intensity, -1, 1) * ForceRate
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
	// Pico
	vendorId1  = 0x2e8a
	productId1 = 0x000a
	// Options
	vendorId2  = 0x2341
	productId2 = 0x8036
)

var (
	alert = js.Global().Get("alert")
	hid   = js.Global().Get("navigator").Get("hid")
)

func (g *Gamepad) rxInputReport1(this js.Value, args []js.Value) any {
	ev := args[0]
	id := ev.Get("reportId").Int()
	b := JS2Bytes(ev.Get("data"))
	switch id {
	case 1:
		axises := []int16{
			int16(binary.LittleEndian.Uint16(b[0:2])), // Axis0
		}
		g.leftStickX = dprec.Clamp(float64(axises[0])/32767, float64(-1), float64(1))
		//log.Printf("rx: %d:%x/%v", id, buttons, axises)
	default:
		log.Printf("rx: %x/%x", id, b)
	}
	return nil
}

func (g *Gamepad) rxInputReport2(this js.Value, args []js.Value) any {
	ev := args[0]
	id := ev.Get("reportId").Int()
	b := JS2Bytes(ev.Get("data"))
	switch id {
	case 3:
		//buttons := b[0:1]
		pad := b[1] & 0x0f
		axises := []uint16{
			binary.LittleEndian.Uint16(b[2:4]),  // Axis0: Side
			binary.LittleEndian.Uint16(b[4:6]),  // Axis1: Throttle
			binary.LittleEndian.Uint16(b[6:8]),  // Axis2: Brake
			binary.LittleEndian.Uint16(b[8:10]), // Axis3: Clutch
		}
		side := int(axises[0]) - 10000
		if side < 0 {
			side = 0
		}
		throttle := int(axises[1]) - 8000
		if throttle < 0 {
			throttle = 0
		}
		brake := int(axises[2]) - 8000
		if brake < 0 {
			brake = 0
		}
		clutch := int(axises[3]) - 8000
		if clutch < 0 {
			clutch = 0
		}
		g.leftTrigger = dprec.Clamp(float64(brake)/40000, float64(0), float64(1))
		g.rightTrigger = dprec.Clamp(float64(throttle)/50000, float64(0), float64(1))
		g.leftStickY = dprec.Clamp(float64(side)/30000, float64(0), float64(1))
		g.forwardButton = pad == 0
		g.backButton = pad == 4
		g.actionUpButton = dprec.Clamp(float64(clutch)/40000, float64(0), float64(1)) > 0.5
		//log.Printf("rx: %d:%x/%d/%d %x", id, pad, axises[3], axises[1], b)
	default:
		log.Printf("rx: %x/%x", id, b)
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
			return args[0].Get("vendorId").Int() == vendorId1 && args[0].Get("productId").Int() == productId1
		})
		dev1 := devices.Call("find", fn)
		fn.Release()
		if dev1.IsNull() || dev1.IsUndefined() {
			alert.Invoke("device not found: dev1")
			g.device = js.Null()
			return
		}
		fn = js.FuncOf(func(this js.Value, args []js.Value) any {
			return args[0].Get("vendorId").Int() == vendorId2 && args[0].Get("productId").Int() == productId2
		})
		dev2 := devices.Call("find", fn)
		fn.Release()
		if dev2.IsNull() || dev2.IsUndefined() {
			alert.Invoke("device not found: dev2")
			g.device2 = js.Null()
			return
		}
		if !dev1.Get("opened").Bool() {
			if _, err := Await(dev1.Call("open")); err != nil {
				alert.Invoke(err.Error())
				return
			}
		}
		if !dev2.Get("opened").Bool() {
			if _, err := Await(dev2.Call("open")); err != nil {
				alert.Invoke(err.Error())
				return
			}
		}
		dev1.Call("addEventListener", "inputreport", js.FuncOf(g.rxInputReport1))
		dev2.Call("addEventListener", "inputreport", js.FuncOf(g.rxInputReport2))
		Await(dev1.Call("sendReport", 0x0c, Bytes2JS([]byte{0x04})))
		Await(dev1.Call("sendReport", 0x0c, Bytes2JS([]byte{0x03})))
		Await(dev1.Call("sendReport", 0x0c, Bytes2JS([]byte{0x01})))
		Await(dev1.Call("sendReport", 0x00, Bytes2JS([]byte{0x01, 0x00, 0x00})))
		Await(dev1.Call("sendReport", 0x01, Bytes2JS([]byte{
			0x01, 0x01, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0x04, 0x3f,
			0x00, 0x00, 0x00, 0x00, 0x00,
		})))
		Await(dev1.Call("sendReport", 0x0a, Bytes2JS([]byte{0x01, 0x01, 0x01})))
		g.pulse = make(chan float64, 16)
		go func() {
			for v := range g.pulse {
				m := int16(dprec.Clamp(v*32767, float64(-32767), float64(32767)))
				Await(dev1.Call("sendReport", 0x05, Bytes2JS([]byte{0x01, byte(m & 0xff), byte(m >> 8)})))
				Await(dev1.Call("sendReport", 0x01, Bytes2JS([]byte{
					0x01, 0x01, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0x04, 0x3f,
					0x00, 0x00, 0x00, 0x00, 0x00,
				})))
				Await(dev1.Call("sendReport", 0x0a, Bytes2JS([]byte{0x01, 0x01, 0x01})))
			}
		}()
		g.device = dev1
		g.device2 = dev2
		log.Println("connect:", g.device.Get("productName"))
		log.Println("connect:", g.device2.Get("productName"))
	}()
}

func GetGamepad() js.Value {
	devices, err := Await(hid.Call("getDevices"))
	if err != nil {
		alert.Invoke(err.Error())
		return js.Null()
	}
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		return args[0].Get("vendorId").Int() == vendorId1 && args[0].Get("productId").Int() == productId1
	})
	dev := devices.Call("find", fn)
	fn.Release()
	if dev.IsNull() || dev.IsUndefined() {
		return js.Null()
	}
	return dev
}

func GamepadConnect() {
	dev := GetGamepad()
	if dev.IsNull() {
		devices, err := Await(hid.Call("requestDevice", map[string]any{
			"filters": []any{
				map[string]any{"productId": productId1, "vendorId": vendorId1},
			}}))
		if err != nil {
			alert.Invoke(err.Error())
			return
		}
		if devices.Length() > 0 {
			dev = devices.Index(0)
		}
	}
	if dev.IsNull() {
		alert.Invoke("No device found")
		return
	}
	devices, err := Await(hid.Call("getDevices"))
	if err != nil {
		alert.Invoke(err.Error())
		return
	}
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		return args[0].Get("vendorId").Int() == vendorId2 && args[0].Get("productId").Int() == productId2
	})
	dev2 := devices.Call("find", fn)
	fn.Release()
	if dev2.IsNull() || dev2.IsUndefined() {
		devices, err := Await(hid.Call("requestDevice", map[string]any{
			"filters": []any{
				map[string]any{"productId": productId2, "vendorId": vendorId2},
			}}))
		if err != nil {
			alert.Invoke(err.Error())
			return
		}
		if devices.Length() > 0 {
			dev2 = devices.Index(0)
		}
	}
	if dev2.IsNull() || dev2.IsUndefined() {
		alert.Invoke("No device found")
		return
	}
	log.Println(dev.Get("productName").String())
	log.Println(dev2.Get("productName").String())
}
