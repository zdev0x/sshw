package sshw

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/atrox/homedir"
	"github.com/kevinburke/ssh_config"
	"github.com/zdev0x/sshw/crypto"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type Node struct {
	Name              string           `yaml:"name" json:"name"`
	Alias             string           `yaml:"alias,omitempty" json:"alias,omitempty"`
	Host              string           `yaml:"host" json:"host"`
	User              string           `yaml:"user,omitempty" json:"user,omitempty"`
	Port              int              `yaml:"port,omitempty" json:"port,omitempty"`
	KeyPath           string           `yaml:"keypath,omitempty" json:"keypath,omitempty"`
	Passphrase        string           `yaml:"passphrase,omitempty" json:"passphrase,omitempty"`
	Password          string           `yaml:"password,omitempty" json:"password,omitempty"`
	IsEncrypted       bool             `yaml:"is_encrypted,omitempty" json:"is_encrypted,omitempty"`
	CallbackShells    []*CallbackShell `yaml:"callback-shells,omitempty" json:"callback-shells,omitempty"`
	Children          []*Node          `yaml:"children,omitempty" json:"children,omitempty"`
	Jump              []*Node          `yaml:"jump,omitempty" json:"jump,omitempty"`
	MaskHost          bool             `yaml:"mask_host,omitempty" json:"mask_host,omitempty"`
	ShowHost          bool             `yaml:"show_host,omitempty" json:"show_host,omitempty"`
	EnableLoginMarker bool             `yaml:"enable_login_marker,omitempty" json:"enable_login_marker,omitempty"`
}

type CallbackShell struct {
	Cmd   string        `yaml:"cmd" json:"cmd"`
	Delay time.Duration `yaml:"delay" json:"delay"`
}

func (n *Node) String() string {
	return n.Name
}

func (n *Node) user() string {
	if n.User == "" {
		return "root"
	}
	return n.User
}

func (n *Node) port() int {
	if n.Port <= 0 {
		return 22
	}
	return n.Port
}

func (n *Node) password() ssh.AuthMethod {
	if n.Password == "" {
		return nil
	}
	return ssh.Password(n.Password)
}

func (n *Node) alias() string {
	return n.Alias
}

var (
	config []*Node
)

func GetConfig() []*Node {
	return config
}

func LoadConfig(password []byte, configPath string) error {
	var b []byte
	var err error

	// 如果指定了配置文件路径，直接加载
	if configPath != "" {
		b, err = ioutil.ReadFile(configPath)
		if err != nil {
			return err
		}
	} else {
		// 否则按默认顺序加载
		b, err = LoadConfigBytes(".sshw", ".sshw.yml", ".sshw.yaml", ".sshw.json")
		if err != nil {
			return err
		}
	}

	var c []*Node
	// 尝试解析为 YAML
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		// 如果 YAML 解析失败，尝试解析为 JSON
		err = json.Unmarshal(b, &c)
		if err != nil {
			return fmt.Errorf("failed to parse config file: %v", err)
		}
	}

	// 如果提供了密码，尝试解密
	if password != nil {
		// 解密所有节点
		for _, node := range c {
			if err := node.DecryptFields(password); err != nil {
				return fmt.Errorf("failed to decrypt config: %v", err)
			}
		}
	}

	config = c
	return nil
}

func LoadSshConfig() error {
	u, err := user.Current()
	if err != nil {
		l.Error(err)
		return nil
	}
	f, _ := os.Open(path.Join(u.HomeDir, ".ssh/config"))
	cfg, _ := ssh_config.Decode(f)
	var nc []*Node
	for _, host := range cfg.Hosts {
		alias := fmt.Sprintf("%s", host.Patterns[0])
		hostName, err := cfg.Get(alias, "HostName")
		if err != nil {
			return err
		}
		if hostName != "" {
			port, _ := cfg.Get(alias, "Port")
			if port == "" {
				port = "22"
			}
			var c = new(Node)
			c.Name = alias
			c.Alias = alias
			c.Host = hostName
			c.User, _ = cfg.Get(alias, "User")
			c.Port, _ = strconv.Atoi(port)
			keyPath, _ := cfg.Get(alias, "IdentityFile")
			c.KeyPath, _ = homedir.Expand(keyPath)
			nc = append(nc, c)
			// fmt.Println(c.Alias, c.Host, c.User, c.Port, c.KeyPath)
		}
	}
	config = nc
	return nil
}

func LoadConfigBytes(names ...string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	// homedir
	for i := range names {
		sshw, err := ioutil.ReadFile(path.Join(u.HomeDir, names[i]))
		if err == nil {
			return sshw, nil
		}
	}
	// relative
	for i := range names {
		sshw, err := ioutil.ReadFile(names[i])
		if err == nil {
			return sshw, nil
		}
	}
	return nil, err
}

// DecryptFields 解密加密的字段
func (n *Node) DecryptFields(key []byte) error {
	if !n.IsEncrypted {
		return nil
	}

	if n.Password != "" {
		decrypted, err := crypto.Decrypt(n.Password, key)
		if err != nil {
			return fmt.Errorf("failed to decrypt password: %v", err)
		}
		n.Password = string(decrypted)
	}

	if n.Passphrase != "" {
		decrypted, err := crypto.Decrypt(n.Passphrase, key)
		if err != nil {
			return fmt.Errorf("failed to decrypt passphrase: %v", err)
		}
		n.Passphrase = string(decrypted)
	}

	n.IsEncrypted = false

	// 递归处理子节点
	for _, child := range n.Children {
		if err := child.DecryptFields(key); err != nil {
			return err
		}
	}

	// 递归处理跳转节点
	for _, jump := range n.Jump {
		if err := jump.DecryptFields(key); err != nil {
			return err
		}
	}

	return nil
}

// EncryptFields 加密敏感字段
func (n *Node) EncryptFields(key []byte) error {
	if n.IsEncrypted {
		return nil
	}

	if n.Password != "" {
		encrypted, err := crypto.Encrypt([]byte(n.Password), key)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %v", err)
		}
		n.Password = encrypted
	}

	if n.Passphrase != "" {
		encrypted, err := crypto.Encrypt([]byte(n.Passphrase), key)
		if err != nil {
			return fmt.Errorf("failed to encrypt passphrase: %v", err)
		}
		n.Passphrase = encrypted
	}

	n.IsEncrypted = true

	// 递归处理子节点
	for _, child := range n.Children {
		if err := child.EncryptFields(key); err != nil {
			return err
		}
	}

	// 递归处理跳转节点
	for _, jump := range n.Jump {
		if err := jump.EncryptFields(key); err != nil {
			return err
		}
	}

	return nil
}

// SaveConfig 保存配置到文件
func SaveConfig(nodes []*Node, configPath string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	var data []byte
	// 根据文件扩展名决定保存格式
	if strings.HasSuffix(configPath, ".json") {
		data, err = json.MarshalIndent(nodes, "", "  ")
	} else {
		data, err = yaml.Marshal(nodes)
	}
	if err != nil {
		return err
	}

	// 如果指定了配置文件路径，保存到指定路径
	if configPath != "" {
		return ioutil.WriteFile(configPath, data, 0600)
	}

	// 否则保存到默认路径
	defaultPath := path.Join(u.HomeDir, ".sshw.yml")
	return ioutil.WriteFile(defaultPath, data, 0600)
}

// IsConfigEncrypted 检查配置是否加密
func IsConfigEncrypted(configPath string) (bool, error) {
	var b []byte
	var err error

	// 如果指定了配置文件路径，直接加载
	if configPath != "" {
		b, err = ioutil.ReadFile(configPath)
		if err != nil {
			return false, err
		}
	} else {
		// 否则按默认顺序加载
		b, err = LoadConfigBytes(".sshw", ".sshw.yml", ".sshw.yaml", ".sshw.json")
		if err != nil {
			return false, err
		}
	}

	var c []*Node
	// 尝试解析为 YAML
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		// 如果 YAML 解析失败，尝试解析为 JSON
		err = json.Unmarshal(b, &c)
		if err != nil {
			return false, fmt.Errorf("failed to parse config file: %v", err)
		}
	}

	// 检查是否有加密的配置
	for _, node := range c {
		if node.IsEncrypted {
			return true, nil
		}
	}
	return false, nil
}

// GetMaskedHost 获取脱敏后的host
func (n *Node) GetMaskedHost() string {
	// 如果ShowHost为false，返回空字符串
	if !n.ShowHost {
		return ""
	}

	// 如果MaskHost为false，返回原始host
	if !n.MaskHost {
		return n.Host
	}

	// 检查是否是IP地址
	ip := net.ParseIP(n.Host)
	if ip != nil {
		// 如果是IP地址，保留前两段和最后一段，中间用*替代
		parts := strings.Split(n.Host, ".")
		if len(parts) == 4 {
			return fmt.Sprintf("%s.%s.*.%s", parts[0], parts[1], parts[3])
		}
		return n.Host
	}

	// 如果是域名，保留第一个字符和最后一个字符，中间用*替代
	parts := strings.Split(n.Host, ".")
	if len(parts) >= 2 {
		// 处理主域名部分
		domain := parts[len(parts)-2]
		if len(domain) > 2 {
			maskedDomain := fmt.Sprintf("%s*%s", domain[:1], domain[len(domain)-1:])
			// 重建完整域名
			parts[len(parts)-2] = maskedDomain
			return strings.Join(parts, ".")
		}
	}
	return n.Host
}
