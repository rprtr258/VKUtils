package text

import (
	"bytes"
	fio "io"
	"io/fs"
	"os"
	"testing"

	"github.com/rprtr258/vk-utils/flow/fun"
	i "github.com/rprtr258/vk-utils/flow/result"
	"github.com/rprtr258/vk-utils/flow/stream"
	"github.com/stretchr/testify/assert"
)

const exampleText = `
Line 2
Line 30
`

func TestTextStream(t *testing.T) {
	data := []byte(exampleText)
	r := bytes.NewReader(data)
	strings := ReadLines(r)
	lens := stream.Map(strings, func(s string) int { return len(s) })
	lensSlice := stream.CollectToSlice(lens)
	assert.ElementsMatch(t, []int{0, 6, 7}, lensSlice)
	stream10_12 := stream.FromMany(10, 11, 12)
	stream20_24 := stream.Map(stream10_12, func(i int) int { return i * 2 })
	res := stream.CollectToSlice(stream20_24)
	assert.Equal(t, []int{20, 22, 24}, res)
}

func TestTextStream2(t *testing.T) {
	data := []byte(exampleText)
	reader := bytes.NewReader(data)
	chunks := ReadByteChunks(reader, 3)
	rows := SplitBySeparator(chunks, '\n')
	strings := stream.Map(rows, func(x []byte) string { return string(x) })
	lens := stream.Map(strings, func(s string) int { return len(s) })
	lensSlice := stream.CollectToSlice(lens)
	assert.ElementsMatch(t, []int{0, 6, 7}, lensSlice)
	stream10_12 := stream.FromMany(10, 11, 12)
	stream20_24 := stream.Map(stream10_12, func(i int) int { return i * 2 })
	res := stream.CollectToSlice(stream20_24)
	assert.Equal(t, []int{20, 22, 24}, res)
}

func TestFile(t *testing.T) {
	path := t.TempDir() + "/hello.txt"
	content := "hello"
	err := os.WriteFile(path, []byte(content), fs.ModePerm)
	assert.NoError(t, err)
	contentIO := i.FlatMap(OpenFile(path), func(f *os.File) i.Result[string] {
		return i.Eval(func() (str string, err error) {
			var bytes []byte
			bytes, err = fio.ReadAll(f)
			if err == nil {
				str = string(bytes)
			}
			return
		})
	})
	str := contentIO.Unwrap()
	assert.Equal(t, content, str)
}

func TestTextStreamWrite(t *testing.T) {
	data := []byte(exampleText)
	r := bytes.NewReader(data)
	strings := ReadLines(r)
	lens := stream.Map(strings, func(s string) int { return len(s) })
	lensAsString := stream.Map(lens, fun.ToString[int])
	w := bytes.NewBuffer([]byte{})
	WriteLines(w, lensAsString)
	assert.Equal(t, `0
6
7
`, w.String())
}
