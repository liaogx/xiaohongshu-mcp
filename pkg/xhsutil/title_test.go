// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xhsutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcTitleLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "空字符串", input: "", want: 0},
		{name: "纯中文", input: "你好世界", want: 4},
		{name: "纯英文", input: "hello", want: 3},
		{name: "纯数字", input: "12345", want: 3},
		{name: "中英混合-OOTD穿搭分享", input: "OOTD穿搭分享", want: 6},
		{name: "20个中文字刚好上限", input: "一二三四五六七八九十一二三四五六七八九十", want: 20},
		{name: "40个英文字母等于20", input: "abcdefghijklmnopqrstuvwxyzabcdefghijklmn", want: 20},
		{name: "单个emoji", input: "😀", want: 2},
		{name: "中文加emoji", input: "今天好开心😀", want: 7},
		{name: "奇数个英文字母向上取整", input: "a", want: 1},
		{name: "两个英文字母", input: "ab", want: 1},
		{name: "三个英文字母", input: "abc", want: 2},
		{name: "全角符号", input: "！？", want: 2},
		{name: "半角符号", input: "!?", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcTitleLength(tt.input))
		})
	}
}
