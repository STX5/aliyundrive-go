package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fs Fs

func setup(t *testing.T) context.Context {
	cb, err := ioutil.ReadFile("../../../.config")
	require.NoError(t, err)
	var config Config
	err = json.Unmarshal(cb, &config)
	require.NoError(t, err)
	ctx := context.Background()
	fs, err = NewFs(ctx, &config)
	require.NoError(t, err)
	return ctx
}

func TestIntegration(t *testing.T) {
	ctx := setup(t)
	info, err := fs.About(ctx)
	require.NoError(t, err)
	fmt.Printf("%#v\n", info)
	testRootNodeId, err := fs.CreateFolderRecursively(ctx, "/")
	require.NoError(t, err)
	childNodeId, err := fs.CreateFolder(ctx, Node{Name: "测试", ParentId: testRootNodeId})
	require.NoError(t, err)
	{
		fd, err := os.Open("../../../assets/rapid_upload.js")
		require.NoError(t, err)
		info, err := fd.Stat()
		require.NoError(t, err)
		nodeId, err := fs.CreateFile(ctx, Node{Name: "rapid_upload.js", ParentId: childNodeId, Size: info.Size()}, fd)
		defer fs.Remove(ctx, childNodeId)
		require.NoError(t, err)
		lNodes, err := fs.ListAll(ctx, childNodeId)
		require.NoError(t, err)
		fmt.Printf("ListAll result: %s\n", lNodes)
		node, err := fs.Get(ctx, nodeId)
		require.NoError(t, err)
		fmt.Printf("node: %s\n", node)

		shareID, sharePwd, expiration, err := fs.CreateShareLink(ctx, []Node{*node}, "1234", Hour*24)
		require.NoError(t, err)
		fmt.Printf("shareID: %s; sharePwd: %s; expire at: %s\n", shareID, sharePwd, expiration)
		shareToken, err := fs.GetShareToken(ctx, sharePwd, shareID)
		fmt.Printf("shareToken: %s", shareToken)
		require.NoError(t, err)
		_, _, _, fileID, err := fs.GetShareInfo(ctx, shareID)
		require.NoError(t, err)
		fmt.Println(fileID)
		Expiration, Creator, err := fs.GetShareLinkByAnonymous(ctx, shareID)
		require.NoError(t, err)
		fmt.Printf("Expiration: %s; Creator: %s", Expiration, Creator)
		time.Sleep(5 * time.Second)
		SharedFile, nextMarker, err := fs.ListShareLinks(ctx)
		require.NoError(t, err)
		fmt.Printf("SharedFileList: %v; nextMarker: %s\n", SharedFile, nextMarker)
		defer func() {
			err := fs.CancelShareLink(ctx, shareID)
			if err != nil {
				panic(err)
			}

		}()

		nodeId, err = fs.Move(ctx, nodeId, childNodeId, "rapid_upload.2.js")
		require.NoError(t, err)
		file, err := fs.Open(ctx, nodeId, map[string]string{})
		require.NoError(t, err)
		data, err := ioutil.ReadAll(file)
		require.NoError(t, err)
		fmt.Printf("read: %s\n", string(data[:20]))
		folderNode, err := fs.Get(ctx, childNodeId)
		folderNode.Meta = "755"
		_, err = fs.Update(ctx, *folderNode)
		require.NoError(t, err)
		node, err = fs.Get(ctx, childNodeId)
		assert.Equal(t, "755", node.Meta)
		fileNode, err := fs.Get(ctx, nodeId)
		fileNode.Meta = "644"
		fileNode.Name = "rapid_upload.3.js"
		_, err = fs.Update(ctx, *fileNode)
		node, err = fs.Get(ctx, nodeId)
		assert.Equal(t, "644", node.Meta)
		assert.Equal(t, "rapid_upload.3.js", node.Name)
	}
	err = fs.Remove(ctx, childNodeId)
	require.NoError(t, err)
}

func TestSha1(t *testing.T) {
	fd, err := os.Open("../../../assets/rapid_upload.js")
	require.NoError(t, err)
	rd, s, err := CalcSha1(fd)
	assert.Equal(t, "3F4D82D88A0624EFD46D7A5FD06BE2D430C00301", s)
	buf := make([]byte, 4)
	_, _ = rd.Read(buf)
	assert.Equal(t, []byte{0x66, 0x75, 0x6e, 0x63}, buf)
}

func TestCalcProof(t *testing.T) {
	fd, err := os.Open("../../../assets/rapid_upload.js")
	require.NoError(t, err)
	fi, err := fd.Stat()
	require.NoError(t, err)
	fileSize := fi.Size()
	accessToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
	require.NoError(t, err)
	proofCode, err := calcProof(accessToken, fileSize, fd)
	assert.Equal(t, "Cn0KCmZ1bmM=", proofCode)
	buf2 := make([]byte, 4)
	_, _ = fd.Read(buf2)
	assert.Equal(t, []byte{0x66, 0x75, 0x6e, 0x63}, buf2)
}
