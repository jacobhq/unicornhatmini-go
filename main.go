package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/rpi"
)

const (
	CmdSoftReset        = 0xCC
	CmdGlobalBrightness = 0x37
	CmdComPinCtrl       = 0x41
	CmdRowPinCtrl       = 0x42
	CmdWriteDisplay     = 0x80
	CmdReadDisplay      = 0x81
	CmdSystemCtrl       = 0x35
	CmdScrollCtrl       = 0x20

	Cols = 17
	Rows = 7

	SpiMaxSpeed = 600 * physic.KiloHertz
)

type Device struct {
	Conn   spi.Conn
	Pin    gpio.PinIO
	Offset int
}

type UnicornHATMini struct {
	rotation    int
	disp        [Cols * Rows][3]uint8
	buf         [28 * 8 * 2]uint8
	lut         [Cols * Rows][3]int
	leftMatrix  Device
	rightMatrix Device
}

func HSVToRGB(h, s, v float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 1.0)
	if h < 0 {
		h += 1.0
	}

	c := v * s
	x := c * (1 - math.Abs(math.Mod(h*6, 2)-1))
	m := v - c

	var r, g, b float64
	switch int(h * 6) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	case 5:
		r, g, b = c, 0, x
	}

	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

func NewUnicornhatmini() (*UnicornHATMini, error) {
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize periph: %v", err)
	}
	var disp [Cols * Rows][3]uint8
	for i := range disp {
		disp[i] = [3]uint8{0, 0, 0}
	}
	var buf [28 * 8 * 2]uint8
	lut := [Cols * Rows][3]int{
		{139, 138, 137}, {223, 222, 221}, {167, 166, 165}, {195, 194, 193},
		{111, 110, 109}, {55, 54, 53}, {83, 82, 81}, {136, 135, 134},
		{220, 219, 218}, {164, 163, 162}, {192, 191, 190}, {108, 107, 106},
		{52, 51, 50}, {80, 79, 78}, {113, 115, 114}, {197, 199, 198},
		{141, 143, 142}, {169, 171, 170}, {85, 87, 86}, {29, 31, 30},
		{57, 59, 58}, {116, 118, 117}, {200, 202, 201}, {144, 146, 145},
		{172, 174, 173}, {88, 90, 89}, {32, 34, 33}, {60, 62, 61},
		{119, 121, 120}, {203, 205, 204}, {147, 149, 148}, {175, 177, 176},
		{91, 93, 92}, {35, 37, 36}, {63, 65, 64}, {122, 124, 123},
		{206, 208, 207}, {150, 152, 151}, {178, 180, 179}, {94, 96, 95},
		{38, 40, 39}, {66, 68, 67}, {125, 127, 126}, {209, 211, 210},
		{153, 155, 154}, {181, 183, 182}, {97, 99, 98}, {41, 43, 42},
		{69, 71, 70}, {128, 130, 129}, {212, 214, 213}, {156, 158, 157},
		{184, 186, 185}, {100, 102, 101}, {44, 46, 45}, {72, 74, 73},
		{131, 133, 132}, {215, 217, 216}, {159, 161, 160}, {187, 189, 188},
		{103, 105, 104}, {47, 49, 48}, {75, 77, 76}, {363, 362, 361},
		{447, 446, 445}, {391, 390, 389}, {419, 418, 417}, {335, 334, 333},
		{279, 278, 277}, {307, 306, 305}, {360, 359, 358}, {444, 443, 442},
		{388, 387, 386}, {416, 415, 414}, {332, 331, 330}, {276, 275, 274},
		{304, 303, 302}, {337, 339, 338}, {421, 423, 422}, {365, 367, 366},
		{393, 395, 394}, {309, 311, 310}, {253, 255, 254}, {281, 283, 282},
		{340, 342, 341}, {424, 426, 425}, {368, 370, 369}, {396, 398, 397},
		{312, 314, 313}, {256, 258, 257}, {284, 286, 285}, {343, 345, 344},
		{427, 429, 428}, {371, 373, 372}, {399, 401, 400}, {315, 317, 316},
		{259, 261, 260}, {287, 289, 288}, {346, 348, 347}, {430, 432, 431},
		{374, 376, 375}, {402, 404, 403}, {318, 320, 319}, {262, 264, 263},
		{290, 292, 291}, {349, 351, 350}, {433, 435, 434}, {377, 379, 378},
		{405, 407, 406}, {321, 323, 322}, {265, 267, 266}, {293, 295, 294},
		{352, 354, 353}, {436, 438, 437}, {380, 382, 381}, {408, 410, 409},
		{324, 326, 325}, {268, 270, 269}, {296, 298, 297},
	}

	leftPort, err := spireg.Open("SPI0.0")
	if err != nil {
		return nil, fmt.Errorf("failed to open SPI0.0: %v", err)
	}
	leftConn, err := leftPort.Connect(SpiMaxSpeed, spi.Mode0, 8)

	if err != nil {
		return nil, fmt.Errorf("SPI0.0 Connect: %v", err)
	}

	rightPort, err := spireg.Open("SPI0.1")
	if err != nil {
		return nil, fmt.Errorf("failed to open SPI0.1: %v", err)
	}
	rightConn, err := rightPort.Connect(SpiMaxSpeed, spi.Mode0, 8)

	if err != nil {
		return nil, fmt.Errorf("SPI0.1 Connect: %v", err)
	}

	leftPin := rpi.P1_24
	rightPin := rpi.P1_26

	leftMatrix := Device{
		Conn:   leftConn,
		Pin:    leftPin,
		Offset: 0,
	}

	rightMatrix := Device{
		Conn:   rightConn,
		Pin:    rightPin,
		Offset: 28 * 8,
	}

	h := &UnicornHATMini{
		rotation:    0,
		disp:        disp,
		buf:         buf,
		lut:         lut,
		leftMatrix:  leftMatrix,
		rightMatrix: rightMatrix,
	}

	devices := []Device{h.leftMatrix, h.rightMatrix}
	for _, d := range devices {
		if err := h.xfer(d, []byte{CmdSoftReset}); err != nil {
			return nil, fmt.Errorf("soft reset failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)

		h.xfer(d, []byte{CmdGlobalBrightness, 0x01})
		h.xfer(d, []byte{CmdScrollCtrl, 0x00})
		h.xfer(d, []byte{CmdSystemCtrl, 0x00})

		writeCmd := append([]byte{CmdWriteDisplay, 0x00}, h.buf[d.Offset:d.Offset+28*8]...)
		h.xfer(d, writeCmd)

		h.xfer(d, []byte{CmdComPinCtrl, 0xff})
		h.xfer(d, []byte{CmdRowPinCtrl, 0xff, 0xff, 0xff, 0xff})
		h.xfer(d, []byte{CmdSystemCtrl, 0x03})
	}

	return h, nil
}

func (h *UnicornHATMini) Shutdown() {
	devices := []Device{h.leftMatrix, h.rightMatrix}
	for _, d := range devices {
		h.xfer(d, []byte{CmdComPinCtrl, 0x00})
		h.xfer(d, []byte{CmdRowPinCtrl, 0x00, 0x00, 0x00, 0x00})
		h.xfer(d, []byte{CmdSystemCtrl, 0x00})
	}
}

func (h *UnicornHATMini) xfer(d Device, data []byte) error {
	if err := d.Pin.Out(gpio.Low); err != nil {
		return fmt.Errorf("error pulling pin low: %v", err)
	}

	if err := d.Conn.Tx(data, nil); err != nil {
		d.Pin.Out(gpio.High)
		return fmt.Errorf("SPI Tx error: %v", err)
	}

	if err := d.Pin.Out(gpio.High); err != nil {
		return fmt.Errorf("error releasing pin: %v", err)
	}

	return nil
}

func (h *UnicornHATMini) SetPixel(x, y int, r, g, b uint8) {
	if x < 0 || x >= Cols || y < 0 || y >= Rows {
		return
	}

	offset := x*Rows + y

	switch h.rotation {
	case 90:
		y = Cols - 1 - y
		offset = y*Rows + x
	case 180:
		x = Cols - 1 - x
		y = Rows - 1 - y
		offset = x*Rows + y
	case 270:
		x = Rows - 1 - x
		offset = y*Rows + x
	}

	if offset >= 0 && offset < len(h.disp) {
		h.disp[offset] = [3]uint8{r >> 2, g >> 2, b >> 2}
	}
}

func (h *UnicornHATMini) SetAll(r, g, b uint8) {
	r >>= 2
	g >>= 2
	b >>= 2

	for i := 0; i < Rows*Cols; i++ {
		h.disp[i] = [3]uint8{r, g, b}
	}
}

func (h *UnicornHATMini) Clear() {
	h.SetAll(0, 0, 0)
}

func (h *UnicornHATMini) SetBrightness(b float64) {
	if b < 0 {
		b = 0
	} else if b > 1 {
		b = 1
	}

	devices := []Device{h.leftMatrix, h.rightMatrix}
	for _, d := range devices {
		brightness := uint8(63 * b)
		h.xfer(d, []byte{CmdGlobalBrightness, brightness})
	}
}

func (h *UnicornHATMini) SetRotation(rotation int) error {
	if rotation != 0 && rotation != 90 && rotation != 180 && rotation != 270 {
		return fmt.Errorf("rotation must be one of 0, 90, 180, 270")
	}
	h.rotation = rotation
	return nil
}

func (h *UnicornHATMini) Show() error {
	for i := 0; i < Cols*Rows; i++ {
		if i >= len(h.lut) {
			continue
		}
		ir, ig, ib := h.lut[i][0], h.lut[i][1], h.lut[i][2]
		r, g, b := h.disp[i][0], h.disp[i][1], h.disp[i][2]

		if ir < len(h.buf) && ig < len(h.buf) && ib < len(h.buf) {
			h.buf[ir] = r
			h.buf[ig] = g
			h.buf[ib] = b
		}
	}

	devices := []Device{h.leftMatrix, h.rightMatrix}
	for _, d := range devices {
		start := d.Offset
		end := start + (28 * 8)
		if end > len(h.buf) {
			end = len(h.buf)
		}
		writeCmd := append([]byte{CmdWriteDisplay, 0x00}, h.buf[start:end]...)
		if err := h.xfer(d, writeCmd); err != nil {
			return err
		}
	}
	return nil
}

func (h *UnicornHATMini) GetShape() (int, int) {
	if h.rotation == 90 || h.rotation == 270 {
		return Rows, Cols
	}
	return Cols, Rows
}

func main() {
	unicorn, err := NewUnicornhatmini()
	if err != nil {
		log.Fatalf("Failed to initialize UnicornHATMini: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down...")
		unicorn.Clear()
		unicorn.Show()
		unicorn.Shutdown()
		os.Exit(0)
	}()

	unicorn.SetBrightness(0.2)

	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentTime := float64(time.Now().UnixNano()) / 1e9

			for y := 0; y < Rows; y++ {
				for x := 0; x < Cols; x++ {
					hue := (currentTime / 4.0) + (float64(x) / float64(Cols*2)) + (float64(y) / float64(Rows))
					r, g, b := HSVToRGB(hue, 1.0, 1.0)
					unicorn.SetPixel(x, y, r, g, b)
				}
			}

			if err := unicorn.Show(); err != nil {
				fmt.Printf("Error showing display: %v\n", err)
			}
		case <-c:
			return
		}
	}
}
