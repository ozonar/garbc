package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"garbage-collector-go/internal/processor"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/joho/godotenv"
)

type FileMoverApp struct {
	window           fyne.Window
	currentDir       string
	fileList         *widget.List
	buttonFrame      *fyne.Container
	selectedID       widget.ListItemID
	lastSelectedID   widget.ListItemID
	lastSelectedTime time.Time
}

func main() {
	// Убедимся, что .env существует и содержит WORK_DIR
	workDir := ensureEnv()

	// Создание приложения Fyne
	a := app.NewWithID("garbage.collector")
	w := a.NewWindow("Garbage Collector 3")
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(600, 500))

	app := &FileMoverApp{
		window:     w,
		currentDir: workDir,
	}

	// Проверка директории
	if !app.checkDirectory() {
		// Запрос выбора директории
		app.chooseDirectory()
	}

	// Автоматическая обработка файлов при запуске
	app.processFilesOnStart()

	// Создание интерфейса
	app.createUI()

	w.ShowAndRun()
}

// ensureEnv гарантирует наличие .env файла с WORK_DIR.
// Возвращает значение WORK_DIR (текущая рабочая директория).
func ensureEnv() string {
	const envFile = ".env"

	// Пытаемся загрузить существующий .env
	if err := godotenv.Load(envFile); err != nil {
		// Файл не существует или нечитаем, создаём новый
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal("Не удалось определить текущую директорию:", err)
		}
		content := fmt.Sprintf("WORK_DIR=%s\n", cwd)
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			log.Fatal("Не удалось создать .env файл:", err)
		}
		log.Println("Создан .env файл с WORK_DIR =", cwd)
		// После создания файла загружаем его
		if err := godotenv.Load(envFile); err != nil {
			log.Fatal("Не удалось загрузить созданный .env файл:", err)
		}
	}

	workDir := os.Getenv("WORK_DIR")
	if workDir == "" {
		// Если WORK_DIR пустой, перезаписываем файл
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal("Не удалось определить текущую директорию:", err)
		}
		content := fmt.Sprintf("WORK_DIR=%s\n", cwd)
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			log.Fatal("Не удалось обновить .env файл:", err)
		}
		log.Println("Обновлён .env файл с WORK_DIR =", cwd)
		workDir = cwd
	}
	return workDir
}

func (app *FileMoverApp) checkDirectory() bool {
	info, err := os.Stat(app.currentDir)
	return err == nil && info.IsDir()
}

func (app *FileMoverApp) chooseDirectory() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			// Если пользователь отменил, завершаем приложение
			os.Exit(1)
			return
		}
		app.currentDir = uri.Path()
		app.processFilesOnStart()
		app.updateInterface()
	}, app.window)
}

func (app *FileMoverApp) processFilesOnStart() {
	if err := processor.ProcessDirectory(app.currentDir); err != nil {
		dialog.ShowError(err, app.window)
	}
}

func (app *FileMoverApp) createUI() {
	// Список файлов
	app.fileList = widget.NewList(
		func() int {
			return len(app.getFileItems())
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			items := app.getFileItems()
			if id < len(items) {
				obj.(*widget.Label).SetText(items[id])
			}
		},
	)
	app.fileList.OnSelected = func(id widget.ListItemID) {
		now := time.Now()
		// Проверка двойного клика
		if app.lastSelectedID == id && now.Sub(app.lastSelectedTime) < 500*time.Millisecond {
			app.openItem(id)
			app.lastSelectedID = -1 // сброс, чтобы не открывать снова
		} else {
			app.selectedID = id
			app.lastSelectedID = id
			app.lastSelectedTime = now
		}
	}
	app.fileList.OnUnselected = func(id widget.ListItemID) {
		app.selectedID = -1
	}

	// Фрейм для кнопок
	app.buttonFrame = container.NewVBox()

	// Основной контейнер
	split := container.NewHSplit(app.fileList, app.buttonFrame)
	split.Offset = 0.7

	app.window.SetContent(split)
	app.updateInterface()
}

func (app *FileMoverApp) getFileItems() []string {
	entries, err := os.ReadDir(app.currentDir)
	if err != nil {
		return []string{}
	}

	var items []string
	for _, entry := range entries {
		name := entry.Name()
		// Пропускаем скрытые, системные и папки с "!"
		if strings.HasPrefix(name, ".") || name == "desktop.ini" || strings.HasPrefix(name, "!") {
			continue
		}
		items = append(items, name)
	}
	return items
}

func (app *FileMoverApp) updateInterface() {
	// Очищаем кнопки
	app.buttonFrame.Objects = nil

	// Кнопка удаления
	deleteBtn := widget.NewButton("Удалить", app.deleteFiles)
	deleteBtn.Importance = widget.HighImportance
	app.buttonFrame.Add(deleteBtn)

	// Кнопки для папок с "!"
	entries, err := os.ReadDir(app.currentDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "!") && entry.IsDir() {
			fullPath := filepath.Join(app.currentDir, name)
			btn := widget.NewButton(name, func(path string) func() {
				return func() { app.moveFiles(path) }
			}(fullPath))
			app.buttonFrame.Add(btn)
		}
	}

	// Обновляем список файлов
	app.fileList.Refresh()
}

func (app *FileMoverApp) moveFiles(targetDir string) {
	if app.selectedID < 0 {
		dialog.ShowInformation("Предупреждение", "Не выбран файл для перемещения", app.window)
		return
	}

	items := app.getFileItems()
	if app.selectedID >= len(items) {
		return
	}
	name := items[app.selectedID]
	src := filepath.Join(app.currentDir, name)
	dst := filepath.Join(targetDir, name)
	if err := os.Rename(src, dst); err != nil {
		dialog.ShowError(fmt.Errorf("не удалось переместить файл %s: %v", name, err), app.window)
		return
	}

	app.selectedID = -1
	app.updateInterface()
}

func (app *FileMoverApp) deleteFiles() {
	if app.selectedID < 0 {
		dialog.ShowInformation("Предупреждение", "Не выбран файл для удаления", app.window)
		return
	}

	items := app.getFileItems()
	if app.selectedID >= len(items) {
		return
	}
	name := items[app.selectedID]
	src := filepath.Join(app.currentDir, name)
	if err := os.Remove(src); err != nil {
		dialog.ShowError(fmt.Errorf("не удалось удалить файл %s: %v", name, err), app.window)
		return
	}

	app.selectedID = -1
	app.updateInterface()
}

// openSystem открывает файл или папку с помощью системного приложения
func openSystem(path string, isDir bool) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		if isDir {
			cmd = exec.Command("explorer", path)
		} else {
			cmd = exec.Command("cmd", "/c", "start", "", path)
		}
	case "darwin":
		cmd = exec.Command("open", path)
	default: // linux и другие
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// openItem открывает выбранный элемент (файл или папку)
func (app *FileMoverApp) openItem(id widget.ListItemID) {
	items := app.getFileItems()
	if id < 0 || id >= len(items) {
		return
	}
	name := items[id]
	fullPath := filepath.Join(app.currentDir, name)

	// Проверяем, является ли путь директорией
	info, err := os.Stat(fullPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("не удалось открыть %s: %v", name, err), app.window)
		return
	}

	if info.IsDir() {
		// Для папки меняем текущую директорию и обновляем интерфейс
		app.currentDir = fullPath
		app.selectedID = -1
		app.updateInterface()
	} else {
		// Для файла открываем с помощью системного ассоциированного приложения
		if err := openSystem(fullPath, false); err != nil {
			dialog.ShowError(fmt.Errorf("не удалось открыть файл %s: %v", name, err), app.window)
		}
	}
}
