package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/bmp"
)

func templateBmpPath(id string) (string, error) {
	dir, err := templatesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".bmp"), nil
}

func writeTemplateBMPFromPNG(id string, pngData []byte) error {
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		return err
	}
	bmpPath, err := templateBmpPath(id)
	if err != nil {
		return err
	}
	f, err := os.Create(bmpPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return bmp.Encode(f, img)
}

func removeTemplateBMP(id string) {
	path, err := templateBmpPath(id)
	if err != nil {
		return
	}
	_ = os.Remove(path)
}

func migrateTemplatesToBMP() error {
	dir, err := templatesDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
			continue
		}
		id := e.Name()[:len(e.Name())-4]
		bmpPath, err := templateBmpPath(id)
		if err != nil {
			continue
		}
		if _, err := os.Stat(bmpPath); err == nil {
			continue
		}
		pngPath := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(pngPath)
		if err != nil {
			continue
		}
		_ = writeTemplateBMPFromPNG(id, data)
	}
	return nil
}
