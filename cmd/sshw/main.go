package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/zdev0x/sshw"
	"github.com/zdev0x/sshw/masterkey"
)

const prev = "-parent-"

var (
	Build                 = "devel"
	showVersion           = flag.Bool("version", false, "show version")
	showHelp              = flag.Bool("help", false, "show help")
	useLocalSSHConfig     = flag.Bool("s", false, "use local ssh config '~/.ssh/config'")
	encryptConfig         = flag.Bool("encrypt", false, "encrypt configuration file")
	decryptConfig         = flag.Bool("decrypt", false, "decrypt configuration file")
	checkEncryptionStatus = flag.Bool("check", false, "check configuration file encryption status")
	setMasterPassword     = flag.Bool("set-master-password", false, "set master password")
	changeMasterPassword  = flag.Bool("change-master-password", false, "change master password")
	removeMasterPassword  = flag.Bool("remove-master-password", false, "remove master password")
	configFile            = flag.String("config", "", "specify configuration file path")

	log = sshw.GetLogger()

	templates = &promptui.SelectTemplates{
		Label:    "✨ {{ . | green}}",
		Active:   "➤ {{ .Name | cyan  }}{{if .Alias}}({{.Alias | yellow}}){{end}}{{if .ShowHost}}{{if .Host}}{{if .User}}{{` ` | faint}}{{.User | faint}}{{`@` | faint}}{{end}}{{.GetMaskedHost | faint}}{{end}}{{end}}",
		Inactive: "  {{.Name | faint}}{{if .Alias}}({{.Alias | faint}}){{end}}{{if .ShowHost}}{{if .Host}}{{if .User}}{{` ` | faint}}{{.User | faint}}{{`@` | faint}}{{end}}{{.GetMaskedHost | faint}}{{end}}{{end}}",
	}
)

func findAlias(nodes []*sshw.Node, nodeAlias string) *sshw.Node {
	for _, node := range nodes {
		if node.Alias == nodeAlias {
			return node
		}
		if len(node.Children) > 0 {
			return findAlias(node.Children, nodeAlias)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *showHelp {
		flag.Usage()
		return
	}

	if *showVersion {
		fmt.Println("sshw - ssh client wrapper for automatic login")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}

	// 处理主密码管理命令
	if *setMasterPassword {
		if err := masterkey.SetMasterPassword(); err != nil {
			log.Error("Failed to set master password:", err)
			os.Exit(1)
		}
		return
	}

	if *changeMasterPassword {
		if err := masterkey.ChangeMasterPassword(); err != nil {
			log.Error("Failed to change master password:", err)
			os.Exit(1)
		}
		return
	}

	if *removeMasterPassword {
		if err := masterkey.RemoveMasterPassword(); err != nil {
			log.Error("Failed to remove master password:", err)
			os.Exit(1)
		}
		return
	}

	// 处理加密相关命令
	if *encryptConfig || *decryptConfig || *checkEncryptionStatus {
		handleEncryptionCommands()
		return
	}

	if *useLocalSSHConfig {
		err := sshw.LoadSshConfig()
		if err != nil {
			log.Error("load ssh config error", err)
			os.Exit(1)
		}
	} else {
		// 检查配置是否加密
		encrypted, err := sshw.IsConfigEncrypted(*configFile)
		if err != nil {
			log.Error("Failed to check config encryption status:", err)
			os.Exit(1)
		}

		var password []byte
		if encrypted {
			// 如果配置已加密，需要获取主密码
			password, err = masterkey.GetMasterPassword()
			if err != nil {
				log.Error("Failed to get master password:", err)
				os.Exit(1)
			}
		}

		// 加载配置
		err = sshw.LoadConfig(password, *configFile)
		if err != nil {
			log.Error("load config error", err)
			os.Exit(1)
		}
	}

	// login by alias
	if len(os.Args) > 1 {
		var nodeAlias = os.Args[1]
		var nodes = sshw.GetConfig()
		var node = findAlias(nodes, nodeAlias)
		if node != nil {
			client := sshw.NewClient(node)
			client.Login()
			return
		}
	}

	node := choose(nil, sshw.GetConfig())
	if node == nil {
		return
	}

	client := sshw.NewClient(node)
	client.Login()
}

func handleEncryptionCommands() {
	// 加载配置以检查每个节点的状态
	err := sshw.LoadConfig(nil, *configFile)
	if err != nil {
		log.Error("Failed to load config:", err)
		os.Exit(1)
	}

	nodes := sshw.GetConfig()
	if len(nodes) == 0 {
		fmt.Println("No configuration found")
		return
	}

	if *checkEncryptionStatus {
		// 检查每个节点的加密状态
		allEncrypted := true
		allUnencrypted := true
		for _, node := range nodes {
			if node.IsEncrypted {
				allUnencrypted = false
			} else {
				allEncrypted = false
			}
		}

		if allEncrypted {
			fmt.Println("All configurations are encrypted")
		} else if allUnencrypted {
			fmt.Println("All configurations are not encrypted")
		} else {
			fmt.Println("Configuration is partially encrypted")
		}
		return
	}

	// 检查是否有需要加密的配置
	hasUnencrypted := false
	for _, node := range nodes {
		if !node.IsEncrypted && (node.Password != "" || node.Passphrase != "") {
			hasUnencrypted = true
			break
		}
	}

	// 如果是加密命令，检查是否已经全部加密
	if *encryptConfig && !hasUnencrypted {
		fmt.Println("All configurations are already encrypted")
		return
	}

	// 如果是解密命令，检查是否已经全部解密
	if *decryptConfig {
		allDecrypted := true
		for _, node := range nodes {
			if node.IsEncrypted {
				allDecrypted = false
				break
			}
		}
		if allDecrypted {
			fmt.Println("All configurations are already decrypted")
			return
		}
	}

	// 获取主密码
	password, err := masterkey.GetMasterPassword()
	if err != nil {
		log.Error("Failed to get master password:", err)
		os.Exit(1)
	}

	// 重新加载配置（使用密码）
	err = sshw.LoadConfig(password, *configFile)
	if err != nil {
		log.Error("Failed to load config:", err)
		os.Exit(1)
	}

	nodes = sshw.GetConfig()
	if len(nodes) == 0 {
		log.Error("No configuration found")
		os.Exit(1)
	}

	if *encryptConfig {
		// 加密配置
		for _, node := range nodes {
			if err := node.EncryptFields(password); err != nil {
				log.Error("Failed to encrypt config:", err)
				os.Exit(1)
			}
		}
		// 保存加密后的配置
		if err := sshw.SaveConfig(nodes, *configFile); err != nil {
			log.Error("Failed to save encrypted config:", err)
			os.Exit(1)
		}
		fmt.Println("Configuration encrypted successfully")
	} else if *decryptConfig {
		// 解密配置
		for _, node := range nodes {
			if err := node.DecryptFields(password); err != nil {
				log.Error("Failed to decrypt config:", err)
				os.Exit(1)
			}
		}
		// 保存解密后的配置
		if err := sshw.SaveConfig(nodes, *configFile); err != nil {
			log.Error("Failed to save decrypted config:", err)
			os.Exit(1)
		}
		fmt.Println("Configuration decrypted successfully")
	}
}

func choose(parent, trees []*sshw.Node) *sshw.Node {
	prompt := promptui.Select{
		Label:        "select host",
		Items:        trees,
		Templates:    templates,
		Size:         20,
		HideSelected: true,
		Searcher: func(input string, index int) bool {
			node := trees[index]
			content := fmt.Sprintf("%s %s %s", node.Name, node.User, node.Host)
			if strings.Contains(input, " ") {
				for _, key := range strings.Split(input, " ") {
					key = strings.TrimSpace(key)
					if key != "" {
						if !strings.Contains(content, key) {
							return false
						}
					}
				}
				return true
			}
			if strings.Contains(content, input) {
				return true
			}
			return false
		},
	}
	index, _, err := prompt.Run()
	if err != nil {
		return nil
	}

	node := trees[index]
	if len(node.Children) > 0 {
		first := node.Children[0]
		if first.Name != prev {
			first = &sshw.Node{Name: prev}
			node.Children = append(node.Children[:0], append([]*sshw.Node{first}, node.Children...)...)
		}
		return choose(trees, node.Children)
	}

	if node.Name == prev {
		if parent == nil {
			return choose(nil, sshw.GetConfig())
		}
		return choose(nil, parent)
	}

	return node
}
