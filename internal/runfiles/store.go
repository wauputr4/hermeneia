package runfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const DefaultRoot = "runs"

type Store struct {
	Root string
}

func New(root string) Store {
	if root == "" {
		root = DefaultRoot
	}
	return Store{Root: root}
}

func (s Store) PrepareRun(runID string) error {
	for _, dir := range []string{
		s.RunDir(runID),
		filepath.Join(s.RunDir(runID), "output"),
		s.CarouselOutputDir(runID),
		s.VideoOutputDir(runID),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) RunDir(runID string) string {
	return filepath.Join(s.Root, runID)
}

func (s Store) BriefPath(runID string, version int) string {
	return filepath.Join(s.RunDir(runID), fmt.Sprintf("brief.v%d.json", version))
}

func (s Store) ContentPath(runID string) string {
	return filepath.Join(s.RunDir(runID), "content.json")
}

func (s Store) ResearchPath(runID string) string {
	return filepath.Join(s.RunDir(runID), "research.json")
}

func (s Store) HistoryPath(runID string) string {
	return filepath.Join(s.RunDir(runID), "history.md")
}

func (s Store) CarouselOutputDir(runID string) string {
	return filepath.Join(s.RunDir(runID), "output", "carousel")
}

func (s Store) VideoOutputDir(runID string) string {
	return filepath.Join(s.RunDir(runID), "output", "video")
}

func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func WriteText(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func AppendText(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(text)
	return err
}

func Checksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}
