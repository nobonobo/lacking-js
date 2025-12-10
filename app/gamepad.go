//go:build !diy && !dfp

package app

import (
	"math"
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
	return &Gamepad{
		index: index,

		isDirty:     true,
		isConnected: false,
		isSupported: false,

		deadzoneStick:   0.1,
		deadzoneTrigger: 0.0,
	}
}

type Gamepad struct {
	index int

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
	jsGamepad := g.jsGamepad()
	if jsGamepad.IsUndefined() || jsGamepad.IsNull() {
		return
	}
	jsActuator := jsGamepad.Get("vibrationActuator")
	if jsActuator.IsUndefined() || jsActuator.IsNull() {
		return
	}
	jsActuator.Call("playEffect", "dual-rumble", map[string]any{
		"startDelay":      0,
		"duration":        duration.Milliseconds(),
		"weakMagnitude":   intensity,
		"strongMagnitude": intensity,
	})
}

func (g *Gamepad) markDirty() {
	g.isDirty = true
}

func (g *Gamepad) jsGamepad() js.Value {
	jsGamepads := js.Global().Get("navigator").Call("getGamepads")
	if jsGamepads.IsUndefined() || jsGamepads.IsNull() {
		return js.Null()
	}
	return jsGamepads.Index(g.index)
}

func (g *Gamepad) refresh() {
	if !g.isDirty {
		return
	}
	jsGamepad := g.jsGamepad()
	g.isDirty = false
	g.isConnected = !jsGamepad.IsUndefined() && !jsGamepad.IsNull() && jsGamepad.Get("connected").Bool()
	if g.isConnected {
		g.isSupported = jsGamepad.Get("mapping").String() == "standard"
	} else {
		g.isSupported = false
	}
	if g.isSupported {
		axes := jsGamepad.Get("axes")
		buttons := jsGamepad.Get("buttons")
		g.leftStickX = axes.Index(0).Float()
		g.leftStickY = axes.Index(1).Float()
		g.leftStickButton = buttons.Index(10).Get("pressed").Bool()
		g.rightStickX = axes.Index(2).Float()
		g.rightStickY = axes.Index(3).Float()
		g.rightStickButton = buttons.Index(11).Get("pressed").Bool()
		g.leftBumperButton = buttons.Index(4).Get("pressed").Bool()
		g.leftTrigger = buttons.Index(6).Get("value").Float()
		g.rightBumperButton = buttons.Index(5).Get("pressed").Bool()
		g.rightTrigger = buttons.Index(7).Get("value").Float()
		g.dpadLeftButton = buttons.Index(14).Get("pressed").Bool()
		g.dpadRightButton = buttons.Index(15).Get("pressed").Bool()
		g.dpadUpButton = buttons.Index(12).Get("pressed").Bool()
		g.dpadDownButton = buttons.Index(13).Get("pressed").Bool()
		g.actionLeftButton = buttons.Index(2).Get("pressed").Bool()
		g.actionRightButton = buttons.Index(1).Get("pressed").Bool()
		g.actionUpButton = buttons.Index(3).Get("pressed").Bool()
		g.actionDownButton = buttons.Index(0).Get("pressed").Bool()
		g.forwardButton = buttons.Index(9).Get("pressed").Bool()
		g.backButton = buttons.Index(8).Get("pressed").Bool()
	} else {
		g.leftStickX = 0.0
		g.leftStickY = 0.0
		g.leftStickButton = false
		g.rightStickX = 0.0
		g.rightStickY = 0.0
		g.rightStickButton = false
		g.leftBumperButton = false
		g.leftTrigger = 0.0
		g.rightBumperButton = false
		g.rightTrigger = 0.0
		g.dpadLeftButton = false
		g.dpadRightButton = false
		g.dpadUpButton = false
		g.dpadDownButton = false
		g.actionLeftButton = false
		g.actionRightButton = false
		g.actionUpButton = false
		g.actionDownButton = false
		g.forwardButton = false
		g.backButton = false
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

func GamepadConnect() {}
