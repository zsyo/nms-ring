package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

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

	go p.ringWorker()
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

	// 收集通知信息
	go p.collectRing(text)

	// 输出终端
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

var (
	levelCh    = make(chan int, 20)
	ticker     = time.NewTicker(time.Millisecond * 500)
	levelRegex = regexp.MustCompile(`(?: (SSS|SS\+|S S) |  ([SABCDE])  )`)
)

func (p *Proxy) collectRing(text string) {
	// 当文本中有 "加载完成" 四个字的时候,开启一个新的等级收集器,并在500毫秒后统计收集器中所有等级的最高级进行尝试提醒
	if strings.Contains(text, "加载完成") {
		ticker.Reset(time.Millisecond * 500)
	}

	ls := levelRegex.FindAllStringSubmatch(text, -1)
	if len(ls) > 0 {
		// fmt.Printf("匹配等级数据: %#v \n", ls)
		var maxL int
		for _, l := range ls {
			switch l[2] {
			case "SSS":
				maxL = max(maxL, ring.LevelSSS)
			case "SS+":
				maxL = max(maxL, ring.LevelSSPlus)
			case "S S":
				maxL = max(maxL, ring.LevelSS)
			case "S":
				maxL = max(maxL, ring.LevelS)
			case "A":
				maxL = max(maxL, ring.LevelA)
			case "B":
				maxL = max(maxL, ring.LevelB)
			case "C":
				maxL = max(maxL, ring.LevelC)
			case "D":
				maxL = max(maxL, ring.LevelD)
			case "E":
				maxL = max(maxL, ring.LevelE)
			}
		}
		levelCh <- maxL
	}
}

func (p *Proxy) ringWorker() {
	for range ticker.C {
		var maxL int
		var hasValue bool

		for {
			select {
			case <-p.ctx.Done():
				ticker.Stop()
				close(levelCh)
				return
			case v := <-levelCh:
				if !hasValue || v > maxL {
					maxL = v
				}
				hasValue = true
			default:
				// channel空了
				if hasValue {
					// fmt.Println("最大等级:", maxL)
					ring.Play(maxL)
				}
				goto END
			}
		}
	END:
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
