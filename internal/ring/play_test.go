package ring

import (
	"testing"
	"time"
)

func TestPlay(t *testing.T) {
	Init("E")
	for level := LevelE; level <= LevelSSS; level++ {
		Play(level)
		time.Sleep(time.Second)
	}
}

func TestSetCustomRing(t *testing.T) {
	path := "D:/zephyr/Music/ring.mp3"
	SetCustomRing(path)
	Play(LevelS)
}

func TestGoPlay(t *testing.T) {
	Init("E")
	go Play(LevelS)
	time.Sleep(time.Second)
	Play(LevelSS)
}
