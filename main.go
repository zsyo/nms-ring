package main

import (
	"fmt"
	"os"
	"time"

	"nms-ring/internal/proxy"
	"nms-ring/internal/ring"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println(`使用示列: ./nms-ring.exe "E:\Tool\Crack\无人深空\超级行星探针.exe"`)
		return
	}
	fmt.Println("3秒后将预览提醒铃声...")
	time.Sleep(time.Second * 3)
	ring.Play()
	fmt.Println("提醒铃声预览结束.(如未听到声音,请检查您的设备音量,并重新运行)")

	proxy.Run(os.Args[1])
}
