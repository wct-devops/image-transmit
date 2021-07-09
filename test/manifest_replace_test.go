package test

import (
	"bytes"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"testing"
	"fmt"
)

func TestBlobRepace(t *testing.T) {

	mainifestStr := `
{
       "schemaVersion": 2,
       "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
       "config": {
          "mediaType": "application/octet-stream",
          "size": 2347,
          "digest": "sha256:83702063e552d5b557fbc09de90b665b58cdaf3a8a1b535b1767cc9492a0cc7e"
       },
       "layers": [
          {
             "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
             "size": 675812,
             "digest": "sha256:eeee0535bf3cec7a24bff2c6e97481afa3d37e2cdeff277c57cb5cbdb2fa9e92"
          },
          {
             "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
             "size": 32,
             "digest": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
          },
          {
             "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
             "size": 32,
             "digest": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
          },
          {
             "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
             "size": 9345154,
             "digest": "sha256:f734a990e57bef723dad1419a12b12c11dff94f62c58e66216edd99efb5e903a"
          },
          {
             "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
             "size": 32,
             "digest": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
          }
       ]
    }`

	manifestByte := []byte(mainifestStr)

	bdg,_ := digest.Parse("sha256:eeee0535bf3cec7a24bff2c6e97481afa3d37e2cdeff277c57cb5cbdb2fa9e92")
	b := types.BlobInfo{
		Digest: bdg,
		Size: 675812,
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
	}

	ndg,_ := digest.Parse("sha256:e3b3aa70783d6d4b1f4d59ff0235bfad9a7ba648dab4c2ba748c3436f7b84764")
	n := types.BlobInfo{
		Digest: ndg,
		Size: 699642,
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
	}

	start := bytes.Index(manifestByte, []byte(b.Digest.String()))
	begIdx := bytes.LastIndex(manifestByte[0:start], []byte{'{'} )
	endIdx := bytes.Index(manifestByte[start:], []byte{'}'} )
	oldBytes := manifestByte[begIdx : start + endIdx]
	newBytes := bytes.ReplaceAll(oldBytes, []byte(b.Digest.String()), []byte(n.Digest.String()))
	newBytes =  bytes.ReplaceAll(newBytes, []byte(fmt.Sprintf(": %v", b.Size )), []byte(fmt.Sprintf(": %v", n.Size )))
	manifestByte = bytes.ReplaceAll(manifestByte, oldBytes, newBytes)

	t.Logf("oldBytes: %s", oldBytes)
	t.Logf("newBytes: %s", newBytes)
	t.Logf("manifestByte: %s", manifestByte)

}
