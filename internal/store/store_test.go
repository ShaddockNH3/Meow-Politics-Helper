package store

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return New(t.TempDir())
}

func TestValidateUserID(t *testing.T) {
	s := newTestStore(t)
	valid := []string{"cat", "cat_123", "Cat-Lover", "a"}
	invalid := []string{"", " spaces", "sp ", "路径", "../etc", "a/b", string(make([]byte, 51))}
	for _, id := range valid {
		if err := s.ValidateUserID(id); err != nil {
			t.Errorf("ValidateUserID(%q) 应通过，但报错: %v", id, err)
		}
	}
	for _, id := range invalid {
		if err := s.ValidateUserID(id); err == nil {
			t.Errorf("ValidateUserID(%q) 应拒绝，但通过了", id)
		}
	}
}

func TestPathTraversalBlocked(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Path("../victim", "file.json"); err == nil {
		t.Fatal("非法 userID 应导致 Path 报错")
	}
	path, err := s.Path("gooduser", "../../escape.json")
	if err != nil {
		t.Fatalf("合法请求不应报错: %v", err)
	}
	if filepath.Base(path) != "escape.json" {
		t.Fatalf("文件名应被 Base 清洗，实际: %s", path)
	}
}

func TestSaveLoadJSONRoundTrip(t *testing.T) {
	s := newTestStore(t)
	type data struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	// 文件不存在时 Load 保持零值
	var out data
	if err := s.LoadJSON("u1", "x.json", &out); err != nil {
		t.Fatalf("LoadJSON(不存在) 不应报错: %v", err)
	}
	if out.Name != "" || out.N != 0 {
		t.Fatalf("文件不存在时 target 应保持零值: %+v", out)
	}

	if err := s.SaveJSON("u1", "x.json", data{Name: "喵", N: 7}); err != nil {
		t.Fatalf("SaveJSON 失败: %v", err)
	}
	out = data{}
	if err := s.LoadJSON("u1", "x.json", &out); err != nil {
		t.Fatalf("LoadJSON 失败: %v", err)
	}
	if out.Name != "喵" || out.N != 7 {
		t.Fatalf("往返数据不一致: %+v", out)
	}

	// 覆盖写也应是原子的
	if err := s.SaveJSON("u1", "x.json", data{Name: "b", N: 2}); err != nil {
		t.Fatalf("覆盖 SaveJSON 失败: %v", err)
	}
	out = data{}
	_ = s.LoadJSON("u1", "x.json", &out)
	if out.N != 2 {
		t.Fatalf("覆盖写后数据不一致: %+v", out)
	}
}

func TestLoadJSONBOM(t *testing.T) {
	s := newTestStore(t)
	if err := s.EnsureUserDir("u1"); err != nil {
		t.Fatal(err)
	}
	path, _ := s.Path("u1", "bom.json")
	// 写入带 BOM 的合法 JSON
	if err := os.WriteFile(path, append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"ok":true}`)...), 0o644); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		OK bool `json:"ok"`
	}
	if err := s.LoadJSON("u1", "bom.json", &payload); err != nil {
		t.Fatalf("带 BOM 的文件应能正常读取: %v", err)
	}
	if !payload.OK {
		t.Fatal("BOM 文件内容解析不正确")
	}
}

func TestRenameFileMissing(t *testing.T) {
	s := newTestStore(t)
	if err := s.RenameFile("u1", "nope.json", "2026"); err != nil {
		t.Fatalf("重命名不存在的文件应静默成功: %v", err)
	}
}
