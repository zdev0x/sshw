package sshw

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	DefaultCiphers = []string{
		"aes128-ctr",
		"aes192-ctr",
		"aes256-ctr",
		"aes128-gcm@openssh.com",
		"chacha20-poly1305@openssh.com",
		"arcfour256",
		"arcfour128",
		"arcfour",
		"aes128-cbc",
		"3des-cbc",
		"blowfish-cbc",
		"cast128-cbc",
		"aes192-cbc",
		"aes256-cbc",
	}
)

type Client interface {
	Login()
}

type defaultClient struct {
	clientConfig *ssh.ClientConfig
	node         *Node
}

// 添加新的结构体用于登录标记
type LoginMarker struct {
	Version string
	User    string
	Host    string
	Time    time.Time
}

func genSSHConfig(node *Node) *defaultClient {
	u, err := user.Current()
	if err != nil {
		l.Error(err)
		return nil
	}

	var authMethods []ssh.AuthMethod

	var pemBytes []byte
	if node.KeyPath == "" {
		pemBytes, err = ioutil.ReadFile(filepath.Join(u.HomeDir, ".ssh/id_rsa"))
	} else {
		pemBytes, err = ioutil.ReadFile(node.KeyPath)
	}
	if err != nil {
		l.Error(err)
	} else {
		var signer ssh.Signer
		if node.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(node.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(pemBytes)
		}
		if err != nil {
			l.Error(err)
		} else {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	password := node.password()

	if password != nil {
		authMethods = append(authMethods, password)
	}

	authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := make([]string, 0, len(questions))
		for i, q := range questions {
			fmt.Print(q)
			if echos[i] {
				scan := bufio.NewScanner(os.Stdin)
				if scan.Scan() {
					answers = append(answers, scan.Text())
				}
				err := scan.Err()
				if err != nil {
					return nil, err
				}
			} else {
				b, err := terminal.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return nil, err
				}
				fmt.Println()
				answers = append(answers, string(b))
			}
		}
		return answers, nil
	}))

	config := &ssh.ClientConfig{
		User:            node.user(),
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 10,
	}

	config.SetDefaults()
	config.Ciphers = append(config.Ciphers, DefaultCiphers...)

	return &defaultClient{
		clientConfig: config,
		node:         node,
	}
}

func NewClient(node *Node) Client {
	return genSSHConfig(node)
}

// 添加新的方法用于设置登录标记
func (c *defaultClient) setLoginMarker(client *ssh.Client) error {
	// 设置环境变量
	envVars := []string{
		fmt.Sprintf("SSH_CLIENT_SSHW=true"),
		fmt.Sprintf("SSH_CLIENT_SSHW_VERSION=%s", "1.0.0"), // 使用固定版本号
		fmt.Sprintf("SSH_CLIENT_SSHW_USER=%s", os.Getenv("USER")),
		fmt.Sprintf("SSH_CLIENT_SSHW_HOST=%s", os.Getenv("HOSTNAME")),
	}

	// 构建登录标记命令
	marker := &LoginMarker{
		Version: "1.0.0", // 使用固定版本号
		User:    os.Getenv("USER"),
		Host:    os.Getenv("HOSTNAME"),
		Time:    time.Now(),
	}

	// 构建登录标记命令
	markerCmd := fmt.Sprintf(`
		# 设置环境变量
		%s

		# 创建日志目录
		mkdir -p ~/.sshw_logs

		# 记录登录信息
		echo "[%s] SSHW Login: User=%s, Host=%s, Version=%s" >> ~/.sshw_logs/login.log

		# 设置登录提示符
		if [ -f ~/.bashrc ]; then
			echo 'export PS1="\[\033[1;32m\][SSHW]\[\033[0m\] $PS1"' >> ~/.bashrc
		fi

		# 清理旧日志（保留最近30天）
		find ~/.sshw_logs -name "login.log" -mtime +30 -delete
	`, strings.Join(envVars, "\n"),
		marker.Time.Format("2006-01-02 15:04:05"),
		marker.User,
		marker.Host,
		marker.Version)

	// 执行登录标记命令
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	if err := session.Run(markerCmd); err != nil {
		return fmt.Errorf("failed to set login marker: %v", err)
	}

	return nil
}

func (c *defaultClient) Login() {
	host := c.node.Host
	port := strconv.Itoa(c.node.port())
	jNodes := c.node.Jump

	var client *ssh.Client

	if len(jNodes) > 0 {
		jNode := jNodes[0]
		jc := genSSHConfig(jNode)
		proxyClient, err := ssh.Dial("tcp", net.JoinHostPort(jNode.Host, strconv.Itoa(jNode.port())), jc.clientConfig)
		if err != nil {
			l.Error(err)
			return
		}
		conn, err := proxyClient.Dial("tcp", net.JoinHostPort(host, port))
		if err != nil {
			l.Error(err)
			return
		}
		ncc, chans, reqs, err := ssh.NewClientConn(conn, net.JoinHostPort(host, port), c.clientConfig)
		if err != nil {
			l.Error(err)
			return
		}
		client = ssh.NewClient(ncc, chans, reqs)
	} else {
		client1, err := ssh.Dial("tcp", net.JoinHostPort(host, port), c.clientConfig)
		client = client1
		if err != nil {
			msg := err.Error()
			// use terminal password retry
			if strings.Contains(msg, "no supported methods remain") && !strings.Contains(msg, "password") {
				fmt.Printf("%s@%s's password:", c.clientConfig.User, host)
				var b []byte
				b, err = terminal.ReadPassword(int(syscall.Stdin))
				if err == nil {
					p := string(b)
					if p != "" {
						c.clientConfig.Auth = append(c.clientConfig.Auth, ssh.Password(p))
					}
					fmt.Println()
					client, err = ssh.Dial("tcp", net.JoinHostPort(host, port), c.clientConfig)
				}
			}
		}
		if err != nil {
			l.Error(err)
			return
		}
	}
	defer client.Close()

	l.Infof("connect server ssh -p %d %s@%s version: %s\n", c.node.port(), c.node.user(), host, string(client.ServerVersion()))

	session, err := client.NewSession()
	if err != nil {
		l.Error(err)
		return
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		l.Error(err)
		return
	}
	defer terminal.Restore(fd, state)

	//changed fd to int(os.Stdout.Fd()) becaused terminal.GetSize(fd) doesn't work in Windows
	//refrence: https://github.com/golang/go/issues/20388
	w, h, err := terminal.GetSize(int(os.Stdout.Fd()))

	if err != nil {
		l.Error(err)
		return
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = session.RequestPty("xterm", h, w, modes)
	if err != nil {
		l.Error(err)
		return
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		l.Error(err)
		return
	}

	err = session.Shell()
	if err != nil {
		l.Error(err)
		return
	}

	// then callback
	for i := range c.node.CallbackShells {
		shell := c.node.CallbackShells[i]
		time.Sleep(shell.Delay * time.Millisecond)
		stdinPipe.Write([]byte(shell.Cmd + "\r"))
	}

	// change stdin to user
	go func() {
		_, err = io.Copy(stdinPipe, os.Stdin)
		l.Error(err)
		session.Close()
	}()

	// interval get terminal size
	// fix resize issue
	go func() {
		var (
			ow = w
			oh = h
		)
		for {
			cw, ch, err := terminal.GetSize(fd)
			if err != nil {
				break
			}

			if cw != ow || ch != oh {
				err = session.WindowChange(ch, cw)
				if err != nil {
					break
				}
				ow = cw
				oh = ch
			}
			time.Sleep(time.Second)
		}
	}()

	// send keepalive
	go func() {
		for {
			time.Sleep(time.Second * 10)
			client.SendRequest("keepalive@openssh.com", false, nil)
		}
	}()

	// 在登录成功后设置登录标记
	if c.node.EnableLoginMarker {
		if err := c.setLoginMarker(client); err != nil {
			GetLogger().Error("Failed to set login marker:", err)
		}
	}

	session.Wait()
}
