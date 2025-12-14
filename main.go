package main

import (
	"fmt"
	"os"

	"nms-ring/internal/proxy"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println(`使用示列: ./nms-ring.exe "E:\Tool\Crack\无人深空\超级行星探针.exe"`)
		return
	}
	proxy.Run(os.Args[1])
}
