# unicornhatmini-go

A Go library for the
Pimoroni [Unicorn HAT Mini](https://shop.pimoroni.com/products/unicorn-hat-mini?variant=31657688498259). This is
currently an almost direct translation
of [pimoroni/unicornhatmini-python](https://github.com/pimoroni/unicornhatmini-python), but this library brings the
speed and ease of deployment that come from using the Go programming language.

## Setup and Installation

Enable SPI on your Raspberry Pi:

```
sudo raspi-config nonint do_spi 0
```

Install the module and import normally:

```
go get github.com/jacobhq/unicornhatmini-go@latest
```

---

*For a usage example, see [examples/main.go](examples/main.go).*

This library has only been tested on a Pi 4, so if you experience problems on other models, feel free to open an issue.