package masterkey

import (
	"fmt"
	"syscall"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	serviceName = "sshw"
	userName    = "master"
)

var (
	ErrNotFound = fmt.Errorf("master password not found")
)

// PasswordStore 定义密码存储接口
type PasswordStore interface {
	Get() ([]byte, error)
	Set([]byte) error
	GetHash() ([]byte, error)
	Delete() error
}

// KeyringStore 实现系统密钥环存储
type KeyringStore struct {
	service  string
	username string
}

func NewKeyringStore() *KeyringStore {
	return &KeyringStore{
		service:  serviceName,
		username: userName,
	}
}

func (s *KeyringStore) Get() ([]byte, error) {
	password, err := keyring.Get(s.service, s.username)
	if err != nil {
		return nil, err
	}
	return []byte(password), nil
}

func (s *KeyringStore) Set(password []byte) error {
	return keyring.Set(s.service, s.username, string(password))
}

func (s *KeyringStore) GetHash() ([]byte, error) {
	hash, err := keyring.Get(s.service, s.username+"_hash")
	if err != nil {
		return nil, err
	}
	return []byte(hash), nil
}

func (s *KeyringStore) Delete() error {
	return keyring.Delete(s.service, s.username)
}

// GetPasswordStore 获取密码存储实例
func GetPasswordStore() (PasswordStore, error) {
	// 尝试使用系统密钥环
	store := NewKeyringStore()
	_, err := store.Get()
	if err == nil {
		return store, nil
	}

	// 如果系统密钥环不可用，使用文件存储
	fileStore, err := NewFileStore()
	if err != nil {
		return nil, err
	}
	return fileStore, nil
}

// GetMasterPassword 获取主密码
func GetMasterPassword() ([]byte, error) {
	store, err := GetPasswordStore()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize password store: %v", err)
	}

	// 尝试获取已存储的密码
	password, err := store.Get()
	if err == nil {
		// 如果成功获取到密码，直接返回
		return password, nil
	}

	// 如果没有存储的密码，请求用户输入
	fmt.Print("Enter master password: ")
	password, err = terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %v", err)
	}

	// 验证密码
	hash, err := store.GetHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get password hash: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, password); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	// 存储密码以供后续使用
	if err := store.Set(password); err != nil {
		return nil, fmt.Errorf("failed to store password: %v", err)
	}

	return password, nil
}

// SetMasterPassword 设置主密码
func SetMasterPassword() error {
	store, err := GetPasswordStore()
	if err != nil {
		return fmt.Errorf("failed to initialize password store: %v", err)
	}

	// 检查是否已设置密码
	_, err = store.Get()
	if err == nil {
		return fmt.Errorf("master password already set")
	}

	// 请求用户输入密码
	fmt.Print("Enter new master password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password: %v", err)
	}

	// 生成密码哈希
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %v", err)
	}

	// 存储密码哈希
	if err := store.Set(hash); err != nil {
		return fmt.Errorf("failed to store password hash: %v", err)
	}

	fmt.Println("Master password set successfully")
	return nil
}

// ChangeMasterPassword 修改主密码
func ChangeMasterPassword() error {
	store, err := GetPasswordStore()
	if err != nil {
		return fmt.Errorf("failed to initialize password store: %v", err)
	}

	// 验证当前密码
	_, err = GetMasterPassword()
	if err != nil {
		return fmt.Errorf("failed to verify current password: %v", err)
	}

	// 请求用户输入新密码
	fmt.Print("Enter new master password: ")
	newPassword, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read new password: %v", err)
	}

	// 生成新密码哈希
	hash, err := bcrypt.GenerateFromPassword(newPassword, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate new password hash: %v", err)
	}

	// 存储新密码哈希
	if err := store.Set(hash); err != nil {
		return fmt.Errorf("failed to store new password hash: %v", err)
	}

	fmt.Println("Master password changed successfully")
	return nil
}

// RemoveMasterPassword 删除主密码
func RemoveMasterPassword() error {
	store, err := GetPasswordStore()
	if err != nil {
		return fmt.Errorf("failed to initialize password store: %v", err)
	}

	// 验证当前密码
	_, err = GetMasterPassword()
	if err != nil {
		return fmt.Errorf("failed to verify current password: %v", err)
	}

	// 删除密码
	if err := store.Delete(); err != nil {
		return fmt.Errorf("failed to remove master password: %v", err)
	}

	fmt.Println("Master password removed successfully")
	return nil
}
