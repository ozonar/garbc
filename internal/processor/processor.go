package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProcessDirectory выполняет автоматическую обработку файлов в директории:
// - Удаляет .torrent файлы (перемещает в корзину, если возможно, иначе удаляет)
// - Перемещает изображения в поддиректорию !img, если она существует
func ProcessDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("не удалось прочитать директорию %s: %w", dir, err)
	}

	// Проверяем наличие папки !img
	imgDir := filepath.Join(dir, "!img")
	imgDirExists := false
	if info, err := os.Stat(imgDir); err == nil && info.IsDir() {
		imgDirExists = true
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())

		// Пропускаем скрытые файлы и системные
		if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "desktop.ini" {
			continue
		}

		// Удаление .torrent файлов
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".torrent") && !entry.IsDir() {
			if err := trashFile(fullPath); err != nil {
				fmt.Printf("Не удалось удалить файл %s: %v\n", entry.Name(), err)
			} else {
				fmt.Printf("Удалён .torrent файл: %s\n", entry.Name())
			}
			continue
		}

		// Перемещение изображений
		if imgDirExists && !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if isImageExt(ext) {
				dst := filepath.Join(imgDir, entry.Name())
				if err := os.Rename(fullPath, dst); err != nil {
					fmt.Printf("Не удалось переместить изображение %s: %v\n", entry.Name(), err)
				} else {
					fmt.Printf("Перемещено изображение: %s -> %s\n", entry.Name(), dst)
				}
			}
		}
	}
	return nil
}

// trashFile перемещает файл в корзину (или удаляет, если корзина недоступна)
func trashFile(path string) error {
	return os.Remove(path)
}

func isImageExt(ext string) bool {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	for _, e := range imageExts {
		if ext == e {
			return true
		}
	}
	return false
}
