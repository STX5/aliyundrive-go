package drive

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fs Fs

func setup(t *testing.T) context.Context {
	token := ""
	cb, err := ioutil.ReadFile("../../../.config")
	if err == nil {
		token = string(cb)
	}
	config := &Config{
		RefreshToken: token,
		IsAlbum:      false,
		HttpClient:   &http.Client{},
	}

	ctx := context.Background()
	fs, err = NewFs(ctx, config)
	require.NoError(t, err)
	return ctx
}

func sleep() {
	time.Sleep(time.Millisecond * 200)
}

func TestIntegration(t *testing.T) {
	ctx := setup(t)
	testRootNodeId, err := fs.CreateFolderRecursively(ctx, "/")
	sleep()
	require.NoError(t, err)
	childNodeId, err := fs.CreateFolder(ctx, testRootNodeId, "测试")
	sleep()
	require.NoError(t, err)
	{
		fd, err := os.Open("../../../assets/rapid_upload.js")
		require.NoError(t, err)
		info, err := fd.Stat()
		require.NoError(t, err)
		nodeId, err := fs.CreateFile(ctx, childNodeId, "rapid_upload.js", info.Size(), fd)
		require.NoError(t, err)
		sleep()
		node, err := fs.Get(ctx, nodeId)
		require.NoError(t, err)
		sleep()
		fmt.Printf("node: %s\n", node)
		nodeId, err = fs.Move(ctx, nodeId, childNodeId, "rapid_upload.2.js")
		require.NoError(t, err)
		sleep()
		file, err := fs.Open(ctx, nodeId, map[string]string{})
		require.NoError(t, err)
		data, err := ioutil.ReadAll(file)
		require.NoError(t, err)
		fmt.Printf("read: %s\n", string(data[:20]))
		sleep()
	}
	err = fs.Remove(ctx, childNodeId)
	require.NoError(t, err)
}

func TestSha1(t *testing.T) {
	fd, err := os.Open("1.mp3")
	require.NoError(t, err)
	rd, s, err := CalcSha1(fd)
	assert.Equal(t, "462FD5A7D4B12EE8A88CF0881D811BD224DB79FE", s)
	buf := make([]byte, 4)
	_, _ = rd.Read(buf)
	assert.Equal(t, []byte{0x49, 0x44, 0x33, 0x03}, buf)
}

func TestCalcProof(t *testing.T) {
	fd, err := os.Open("1.mp3")
	accessToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
	fileSize := int64(4087117)
	require.NoError(t, err)
	rd, proofCode, err := calcProof(accessToken, fileSize, fd)
	assert.Equal(t, "dj66UE3TEFM=", proofCode)
	buf2 := make([]byte, 4)
	_, _ = rd.Read(buf2)
	assert.Equal(t, []byte{0x49, 0x44, 0x33, 0x03}, buf2)
}
