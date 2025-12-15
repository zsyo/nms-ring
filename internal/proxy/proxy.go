package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"nms-ring/internal/ring"

	"github.com/UserExistsError/conpty"
)

type Proxy struct {
	cmd string
	ctx context.Context

	pty *conpty.ConPty
}

func (p *Proxy) Run() {
	var err error
	p.pty, err = conpty.Start(p.cmd, conpty.ConPtyDimensions(120, 40))
	if err != nil {
		fmt.Println("启动程序失败:", err)
		return
	}
	defer p.pty.Close()

	go p.readLoop()

	// 等待程序退出
	_, err = p.pty.Wait(p.ctx)
	if err != nil {
		fmt.Println("程序异常退出:", err)
		return
	}
}

func (p *Proxy) readLoop() {
	buf := make([]byte, 4096)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			n, err := p.pty.Read(buf)
			if n > 0 {
				p.handleOutput(buf[:n])
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					fmt.Println("读取 PTY 失败:", err)
				}
				return
			}
		}
	}
}

func (p *Proxy) handleOutput(raw []byte) {
	// fmt.Printf("原始文本: ->%q<-\n", raw)

	text := string(raw)
	// 播放铃声
	if strings.Contains(text, "  S  ") ||
		strings.Contains(text, " S S ") ||
		strings.Contains(text, " SS+ ") ||
		strings.Contains(text, " SSS ") {
		go ring.Play()
	}
	fmt.Print(text)

	// 交互输入
	if strings.Contains(text, "[Y]我同意 [N]不同意:") ||
		strings.Contains(text, "请输入命令:") ||
		strings.Contains(text, "请输入选择:") ||
		strings.Contains(text, "输入Q退出探针:") {

		var input string
		fmt.Scan(&input)
		input += "\r\n"
		_, _ = p.pty.Write([]byte(input))
	}
}

func Run(programPath string) {
	if len(programPath) == 0 {
		fmt.Println("程序路径不能为空")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := Proxy{
		ctx: ctx,
		cmd: programPath,
	}

	p.Run()
}
