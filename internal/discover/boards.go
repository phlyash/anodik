package discover

import (
	"anodik/internal/env"
	"fmt"
	"os"
	"path/filepath"
)

type Board struct {
	Soc  string
	Name string
}

func (b Board) String() string {
	return fmt.Sprintf("%s: %s", b.Soc, b.Name)
}

func socDir(s *env.Settings) string {
	return filepath.Join(s.SDKRoot, "components", "soc")
}

func DiscoverBoards(s *env.Settings) ([]Board, error) {
	socDir := socDir(s)

	entries, err := os.ReadDir(socDir)
	if err != nil {
		return nil, err
	}

	var boards []Board

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		switch entry.Name() {
		case "common", "linker":
			continue
		}

		chipPath := filepath.Join(socDir, entry.Name())
		chipEntries, err := os.ReadDir(chipPath)
		if err != nil {
			continue
		}

		for _, chipEntry := range chipEntries {
			if !chipEntry.IsDir() || chipEntry.Name() != "bsp" {
				continue
			}

			bspPath := filepath.Join(chipPath, chipEntry.Name())
			boardDirs, err := os.ReadDir(bspPath)
			if err != nil {
				continue
			}

			for _, board := range boardDirs {
				if board.IsDir() {
					boards = append(boards, Board{
						Soc:  entry.Name(),
						Name: board.Name(),
					})
				}
			}
		}
	}

	return boards, nil
}
