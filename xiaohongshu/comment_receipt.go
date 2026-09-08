// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/liaogx/xiaohongshu-mcp/cookies"
)

type savedCommentReceipt struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	interactionReceipt
}

func commentReceiptPath(userID, feedID, content string) string {
	dir := os.Getenv("XHS_INTERACTION_STATE_DIR")
	if dir == "" {
		dir = filepath.Join(filepath.Dir(cookies.GetCookiesFilePath()), "state", "interaction-receipts")
	}
	// Account scoped and stable across renewed cookies. No plaintext comment,
	// account identifiers or authentication material are stored in the key.
	key := sha256.Sum256([]byte(userID + "\x00" + feedID + "\x00" + content))
	return filepath.Join(dir, "comment-"+hex.EncodeToString(key[:])+".json")
}

func beginCommentReceipt(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return fmt.Errorf("DUPLICATE_OR_UNCERTAIN_COMMENT: 相同账号、笔记和评论已有提交记录；禁止重发，先核验原回执")
	}
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(savedCommentReceipt{Version: 1, UpdatedAt: time.Now(), interactionReceipt: interactionReceipt{State: "unknown"}}); err != nil {
		return err
	}
	return f.Sync()
}

func finishCommentReceipt(path string, result interactionReceipt) error {
	if result.State == "" {
		result.State = "unknown"
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".receipt-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := json.NewEncoder(f).Encode(savedCommentReceipt{Version: 1, UpdatedAt: time.Now(), interactionReceipt: result}); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
