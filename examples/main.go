package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unicornhatmini"
)

func main() {
	fmt.Println("unicornhatmini-go: Displays a moving concentric rainbow on the Unicorn HAT Mini. Press Ctrl+C to exit.")

	unicorn, err := unicornhatmini_go.NewUnicornhatmini()
	if err != nil {
		log.Fatalf("Failed to initialize UnicornHATMini: %v", err)
	}

	unicorn.SetBrightness(0.1)
	unicorn.SetRotation(0)

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

	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()

	step := 0.0

	for {
		select {
		case <-ticker.C:
			step += 1.0
			for x := 0; x < unicornhatmini_go.Cols; x++ {
				for y := 0; y < unicornhatmini_go.Rows; y++ {
					dx := (math.Sin(step/float64(unicornhatmini_go.Cols)+20) * float64(unicornhatmini_go.Cols)) + float64(unicornhatmini_go.Rows)
					dy := (math.Cos(step/float64(unicornhatmini_go.Rows)) * float64(unicornhatmini_go.Rows)) + float64(unicornhatmini_go.Rows)
					sc := (math.Cos(step/float64(unicornhatmini_go.Rows)) * float64(unicornhatmini_go.Rows)) + float64(unicornhatmini_go.Cols)

					dist := math.Sqrt(math.Pow(float64(x)-dx, 2) + math.Pow(float64(y)-dy, 2))
					hue := dist / sc
					r, g, b := unicornhatmini_go.HSVToRGB(hue, 1.0, 1.0)
					unicorn.SetPixel(x, y, r, g, b)
				}
			}

			if err := unicorn.Show(); err != nil {
				fmt.Printf("Error showing display: %v\n", err)
			}
		}
	}
}
