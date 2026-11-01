// Package store 是用户数据唯一的持久化出口。
//
// 所有用户相关的 JSON 文件读写必须经过本包，统一处理：
//   - 用户 ID 校验与目录隔离（防止路径穿越）
//   - 原子写入（先写临时文件再 rename，避免写一半崩溃损坏数据）
//   - UTF-8 BOM 兼容
package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// userIDPattern 限制用户 ID 只能是字母、数字、下划线和短横线，
// 从根源上杜绝通过 userID 拼出 "../" 之类的路径。
var userIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Store 管理某个基础目录下的用户数据。
type Store struct {
	baseDir string
}

// New 创建一个以 baseDir 为根的用户数据仓库。
func New(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// BaseDir 返回用户数据根目录。
func (s *Store) BaseDir() string { return s.baseDir }

// EnsureDir 确保用户数据根目录存在。
func (s *Store) EnsureDir() error {
	return os.MkdirAll(s.baseDir, 0o755)
}

// ValidateUserID 校验用户 ID：非空、1-50 字符、无首尾空格、只含安全字符。
func (s *Store) ValidateUserID(userID string) error {
	if strings.TrimSpace(userID) != userID || userID == "" || len(userID) > 50 {
		return fmt.Errorf("用户ID长度需为 1-50，且不能包含首尾空格")
	}
	if !userIDPattern.MatchString(userID) {
		return fmt.Errorf("用户ID只能包含字母、数字、下划线和短横线")
	}
	return nil
}

// UserDir 返回用户数据目录路径；userID 不合法时报错。
func (s *Store) UserDir(userID string) (string, error) {
	if err := s.ValidateUserID(userID); err != nil {
		return "", err
	}
	return filepath.Join(s.baseDir, userID), nil
}

// EnsureUserDir 确保用户数据目录存在。
func (s *Store) EnsureUserDir(userID string) error {
	dir, err := s.UserDir(userID)
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// Path 返回用户目录下某个文件的完整路径。
// fileName 会被 filepath.Base 清洗，防止子目录穿越。
func (s *Store) Path(userID, fileName string) (string, error) {
	dir, err := s.UserDir(userID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.Base(fileName)), nil
}

// LoadJSON 读取用户目录下的 JSON 文件并反序列化到 target。
// 文件不存在或为空时保持 target 的零值不变（调用方自行预初始化）。
func (s *Store) LoadJSON(userID, fileName string, target any) error {
	path, err := s.Path(userID, fileName)
	if err != nil {
		return err
	}
	if err := s.EnsureUserDir(userID); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取用户文件 %s 失败: %w", filepath.Base(fileName), err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // 兼容带 BOM 的旧文件
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("解析用户文件 %s 失败: %w", filepath.Base(fileName), err)
	}
	return nil
}

// SaveJSON 将数据序列化后原子写入用户目录下的文件。
func (s *Store) SaveJSON(userID, fileName string, data any) error {
	path, err := s.Path(userID, fileName)
	if err != nil {
		return err
	}
	if err := s.EnsureUserDir(userID); err != nil {
		return err
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 %s 失败: %w", filepath.Base(fileName), err)
	}

	// 原子写：临时文件 + rename
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(fileName)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // rename 成功后 remove 会失败，无妨

	if _, err := tmp.Write(jsonData); err != nil {
		tmp.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("设置文件权限失败: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("保存 %s 失败: %w", filepath.Base(fileName), err)
	}
	return nil
}

// UserFiles 列出用户目录下（不含子目录）的文件名。
func (s *Store) UserFiles(userID string) ([]string, error) {
	dir, err := s.UserDir(userID)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// RenameFile 将用户目录下的文件重命名为 <原名>.<时间戳>.bak（用于清理数据前的备份）。
// 文件不存在时返回 nil。
func (s *Store) RenameFile(userID, fileName, stamp string) error {
	path, err := s.Path(userID, fileName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Rename(path, path+"."+stamp+".bak")
}
