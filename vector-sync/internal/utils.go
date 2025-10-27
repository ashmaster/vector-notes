package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func SplitPath(path string) []string {
	var segments []string
	for i, segment := range strings.Split(path, "/") {
		if i == len(strings.Split(path, "/"))-1 {
			// Last segment (file or dir)
			segments = append(segments, segment)
		} else {
			// Directory segment
			segments = append(segments, segment+"/")
		}
	}
	return segments
}

func CalculateHash(data []byte) string {
	contentHash := sha256.Sum256(data)
	return hex.EncodeToString(contentHash[:])
}
