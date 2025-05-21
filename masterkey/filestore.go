package masterkey

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileStore 提供基于文件的密码存储
type FileStore struct {
	path string
}

// NewFileStore 创建新的文件存储
func NewFileStore() (*FileStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	path := filepath.Join(home, ".sshw-master")
	return &FileStore{path: path}, nil
}

// Get 从文件读取密码哈希
func (s *FileStore) Get() ([]byte, error) {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

// Set 将密码哈希保存到文件
func (s *FileStore) Set(data []byte) error {
	return ioutil.WriteFile(s.path, data, 0600)
}

// GetHash 从文件读取密码哈希
func (s *FileStore) GetHash() ([]byte, error) {
	data, err := ioutil.ReadFile(s.path + "_hash")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

// Delete 删除密码文件
func (s *FileStore) Delete() error {
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(s.path + "_hash"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
