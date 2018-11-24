package visitor

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type FileCallback func(filename string, reader io.Reader) (VisitControl, error)

func Files(owner, repo, sha string, v FileCallback) error {
	// TODO(mattmoor): Maybe this should use this:
	// https://godoc.org/github.com/google/go-github/github#RepositoriesService.GetArchiveLink
	url := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", owner, repo, sha)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)

	// All of the files in the archive should have the following prefix
	prefix := fmt.Sprintf("%s-%s/", repo, sha)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if !strings.HasPrefix(header.Name, prefix) {
			return fmt.Errorf("File without prefix: %s", header.Name)
		}
		if !header.FileInfo().Mode().IsRegular() {
			log.Printf("Ignoring file (not regular): %s", header.Name)
			continue
		}
		stripped := header.Name[len(prefix):]
		if vc, err := v(stripped, tr); err != nil {
			return err
		} else if vc == Break {
			return nil
		}
	}

	return nil
}
