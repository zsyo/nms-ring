package main

import (
	"flag"
	"fmt"
	"time"

	"nms-ring/internal/proxy"
	"nms-ring/internal/ring"
)

func main() {
	// 可选参数
	level := flag.String("l", "S", "level: E D C B A S SS SS+ SSS")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Println(`使用示列: ./nms-ring.exe [-l=LEVEL] "E:\Tool\Crack\无人深空\超级行星探针.exe"`)
		return
	}

	ring.Init(*level)
	if ring.IsCustomRingSet() {
		fmt.Println("3秒后将预览提醒铃声(自定义铃声)...")
	} else {
		fmt.Println("3秒后将预览提醒铃声...")
	}
	time.Sleep(time.Second * 3)
	ring.Play(ring.LevelSSS)
	fmt.Println("提醒铃声预览结束.(如未听到声音,请检查您的设备音量,并重新运行)")

	proxy.Run(args[0])
}
