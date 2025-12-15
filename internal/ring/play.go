package ring

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

//go:embed sounds
var sounds embed.FS

var sampleRate = beep.SampleRate(44100)

func init() {
	speaker.Init(sampleRate, sampleRate.N(time.Second/10))
}

const (
	LevelE = iota
	LevelD
	LevelC
	LevelB
	LevelA
	LevelS
	LevelSS
	LevelSSPlus
	LevelSSS
)

var levelDict = map[string]int{
	"E":   LevelE,
	"D":   LevelD,
	"C":   LevelC,
	"B":   LevelB,
	"A":   LevelA,
	"S":   LevelS,
	"SS":  LevelSS,
	"SS+": LevelSSPlus,
	"SSS": LevelSSS,
}

var ringFiles = map[int]string{
	LevelE:      "sounds/e.ogg",
	LevelD:      "sounds/d.ogg",
	LevelC:      "sounds/c.ogg",
	LevelB:      "sounds/b.ogg",
	LevelA:      "sounds/a.ogg",
	LevelS:      "sounds/s.ogg",
	LevelSS:     "sounds/ss.ogg",
	LevelSSPlus: "sounds/ss+.ogg",
	LevelSSS:    "sounds/sss.ogg",
}

var (
	globalLevel = LevelS // 全局铃声等级
	once        sync.Once
	customBuf   *beep.Buffer
	ringBufs    = make(map[int]*beep.Buffer)
)

func setGlobalLevel(level string) {
	level = strings.ToUpper(level)

	if _, ok := levelDict[level]; !ok {
		fmt.Printf("无效的提示等级: [%s] 支持的等级:E,D,C,B,A,S,SS,SS+,SSS", level)
		os.Exit(1)
	}
	globalLevel = levelDict[level]
}

func Init(level string) {
	once.Do(func() {
		setGlobalLevel(level)

		// 先判断自定义铃声文件是否存在(程序同目录下 'customRing.格式' 文件: 仅支持ogg,wav和mp3格式)
		if customRing, ok := findCustomRing(); ok {
			SetCustomRing(customRing)
			return
		}

		// 如果自定义铃声不存在，则使用内置默认铃声
		for level, file := range ringFiles {
			if level < globalLevel {
				continue
			}

			func() {
				fd, err := sounds.Open(file)
				if err != nil {
					fmt.Println("Error opening ring file:", err)
					os.Exit(1)
				}
				defer fd.Close()

				raw, format, err := vorbis.Decode(fd)
				if err != nil {
					fmt.Println("Error decoding ring:", err)
					os.Exit(1)
				}
				defer raw.Close()

				var streamer beep.Streamer = raw
				if format.SampleRate != sampleRate {
					streamer = beep.Resample(4, format.SampleRate, sampleRate, streamer)
					format.SampleRate = sampleRate
				}

				buf := beep.NewBuffer(format)
				buf.Append(streamer)

				ringBufs[level] = buf
			}()
		}
	})
}

func findCustomRing() (string, bool) {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		os.Exit(1)
	}
	dir := filepath.Dir(exePath)

	// 按优先级顺序
	candidates := []string{
		"customRing.ogg",
		"customRing.wav",
		"customRing.mp3",
	}

	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			return path, true
		}
	}
	return "", false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func SetCustomRing(path string) {
	fd, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening custom ring file:", err)
		os.Exit(1)
	}
	defer fd.Close()

	var (
		raw    beep.StreamSeekCloser
		format beep.Format
	)

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ogg":
		raw, format, err = vorbis.Decode(fd)
	case ".wav":
		raw, format, err = wav.Decode(fd)
	case ".mp3":
		raw, format, err = mp3.Decode(fd)
	default:
		fmt.Println("Unsupported file format:", ext)
		os.Exit(1)
	}
	if err != nil {
		fmt.Println("Error decoding custom ring:", err)
		os.Exit(1)
	}
	defer raw.Close()

	var streamer beep.Streamer = raw
	if format.SampleRate != sampleRate {
		streamer = beep.Resample(4, format.SampleRate, sampleRate, streamer)
		format.SampleRate = sampleRate
	}

	customBuf = beep.NewBuffer(format)
	customBuf.Append(streamer)
}

func IsCustomRingSet() bool {
	return customBuf != nil
}

func Play(level int) {
	if level < globalLevel {
		return
	}

	if IsCustomRingSet() {
		// 直接使用用户自定义铃声
		speaker.PlayAndWait(customBuf.Streamer(0, customBuf.Len()))
		return
	}

	buf, ok := ringBufs[level]
	if !ok {
		fmt.Println("Unsupported level:", level)
		return
	}
	speaker.PlayAndWait(buf.Streamer(0, buf.Len()))
}
