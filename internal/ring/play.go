package ring

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
)

//go:embed ring.mp3
var ringFile []byte

func Play() {
	streamer, format, err := mp3.Decode(io.NopCloser(bytes.NewReader(ringFile)))
	if err != nil {
		fmt.Println("Error decoding ring:", err)
		return
	}
	defer streamer.Close()

	// 初始化扬声器
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// 播放音频并等待完成
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))
	<-done
}
