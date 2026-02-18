package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"nms-ring/internal/ring"

	"github.com/UserExistsError/conpty"
)

type Proxy struct {
	cmd    string
	cancel context.CancelFunc

	restart bool

	pty   *conpty.ConPty
	input map[string]string
}

func (p *Proxy) Run() {
	// 首次运行
	p.run()

	// 重启
	for p.restart {
		p.run()
	}

	close(levelCh)
}

func (p *Proxy) run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.cancel = cancel

	var err error
	p.pty, err = conpty.Start(p.cmd, conpty.ConPtyDimensions(180, 40))
	if err != nil {
		fmt.Println("启动程序失败:", err)
		return
	}
	defer p.pty.Close()

	go p.ringWorker(ctx)
	go p.readLoop(ctx)

	// 等待程序退出
	_, err = p.pty.Wait(ctx)
	if err != nil {
		fmt.Println("程序异常退出:", err)
		return
	}
}

func (p *Proxy) readLoop(ctx context.Context) {
	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
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

	// 程序退出还是重启
	if strings.Contains(text, "输入Q退出探针:") {
		text = strings.ReplaceAll(text, "输入Q退出探针:", "输入Q退出探针或输入R重新监听:")
		fmt.Print(text)

		var input string

	reinput:
		fmt.Scan(&input)

		input = strings.ToLower(input)
		switch input {
		case "q":
			p.restart = false
			input += "\r\n"
			_, _ = p.pty.Write([]byte(input))
		case "r":
			p.restart = true
			p.cancel()
		default:
			fmt.Print("无效输入, 请输入Q退出探针或输入R重新监听:")
			goto reinput
		}
		return
	}

	// 输出终端
	fmt.Print(text)

	// 交互输入
	var option string
	if strings.Contains(text, "[Y]我同意 [N]不同意:") {
		option = "LICENSE"
	} else if strings.Contains(text, "请输入命令:") {
		option = "COMMAND"
	} else if strings.Contains(text, "请输入选择:") {
		option = "MODE"
	}
	if option != "" {
		input, ok := p.input[option]
		if !ok {
			fmt.Scan(&input)
			input += "\r\n"

			p.input[option] = input
		} else {
			time.Sleep(time.Millisecond * 500)
		}
		_, _ = p.pty.Write([]byte(input))
	}
}

var (
	levelCh    = make(chan int, 20)
	ticker     = time.NewTicker(time.Millisecond * 500)
	levelRegex = regexp.MustCompile(`\s+(SSS|SS\+|S S|[SABCDE])\s+\x1b\[m`)
	scoreRegex = regexp.MustCompile(`\s+(\d{1,3})\s+\x1b\[m\s+0x([0-9A-F]+)`)
)

func (p *Proxy) collectRing(text string) {
	// fmt.Printf("文本数据: ->%q<-\n", text)

	// 当文本中有 "加载完成" 四个字的时候,开启一个新的等级收集器,并在500毫秒后统计收集器中所有等级的最高级进行尝试提醒
	if strings.Contains(text, "加载完成") {
		ticker.Reset(time.Millisecond * 500)
	}

	ls := levelRegex.FindAllStringSubmatch(text, -1)
	if len(ls) > 0 {
		// fmt.Printf("匹配等级数据: %#v \n", ls)
		var maxL int
		for _, l := range ls {
			switch l[1] {
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

	ss := scoreRegex.FindAllStringSubmatch(text, -1)
	if len(ss) > 0 {
		// fmt.Printf("匹配分数数据: %#v \n", ss)
		var maxL int
		for _, s := range ss {
			score, _ := strconv.Atoi(s[1])
			level := ring.Score2Level(score)
			maxL = max(maxL, level)
		}
		levelCh <- maxL
	}
}

func (p *Proxy) ringWorker(ctx context.Context) {
	for range ticker.C {
		var maxL int
		var hasValue bool

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
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

	p := Proxy{
		cmd:   programPath,
		input: make(map[string]string),
	}

	p.Run()

	fmt.Println("按任意键退出程序...")
	// 加载 Windows 的键盘处理动态链接库
	h, _ := syscall.LoadLibrary("msvcrt.dll")
	getch, _ := syscall.GetProcAddress(h, "_getch")
	// 调用 _getch，程序会在这里阻塞，直到你按下任意一个按键
	syscall.SyscallN(getch)
}
