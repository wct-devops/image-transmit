package core

import (
	"fmt"

	"github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// ManifestHandler expends the ability of handling manifest list in schema2, but it's not finished yet
// return the digest array of manifests in the manifest list if exist.
func ManifestHandler(m []byte, t string, i *ImageSource) ([]manifest.Manifest, error) {

	var manifestInfoSlice []manifest.Manifest

	if t == manifest.DockerV2Schema2MediaType || t == imgspecv1.MediaTypeImageManifest {
		var manifestInfo manifest.Manifest
		var err error
		if t == imgspecv1.MediaTypeImageManifest {
			manifestInfo, err = manifest.OCI1FromManifest(m)
		} else {
			manifestInfo, err = manifest.Schema2FromManifest(m)
		}
		if err != nil {
			return nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil
	} else if t == manifest.DockerV2Schema1MediaType || t == manifest.DockerV2Schema1SignedMediaType {
		manifestInfo, err := manifest.Schema1FromManifest(m)
		if err != nil {
			return nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil
	} else if t == manifest.DockerV2ListMediaType || t == imgspecv1.MediaTypeImageIndex {
		var digests []imgspecv1.Descriptor
		if t == imgspecv1.MediaTypeImageIndex {
			indexInfo, err := manifest.OCI1IndexFromManifest(m)
			if err != nil {
				return nil, err
			}
			digests = indexInfo.Manifests
		} else {
			listInfo, err := manifest.Schema2ListFromManifest(m)
			if err != nil {
				return nil, err
			}
			for _, elem := range listInfo.Manifests {
				digests = append(digests, imgspecv1.Descriptor{Digest: elem.Digest})
			}
		}

		for _, desc := range digests {
			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &desc.Digest)
			if err != nil {
				return nil, err
			}
			if i.subManifests == nil {
				i.subManifests = make(map[string][]byte)
			}
			i.subManifests[desc.Digest.String()] = manifestByte

			platformSpecManifest, err := ManifestHandler(manifestByte, manifestType, i)
			if err != nil {
				return nil, err
			}

			manifestInfoSlice = append(manifestInfoSlice, platformSpecManifest...)
		}
		return manifestInfoSlice, nil
	}

	return nil, fmt.Errorf("unsupported manifest type: %v", t)
}
