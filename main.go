package main

import (
	"BiomeManager/modules"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"log"
)

func main() {

	// Получение путей к REDkit и проектному имени, но игнорируем workspacePath
	fullPath, _, projectName, err := modules.GetPaths()
	if err != nil {
		log.Fatalf("Ошибка при получении путей: %v", err)
	}
	fmt.Println(fullPath, projectName)
	// Инициализируем приложение
	myApp := app.New()
	myWindow := myApp.NewWindow("Biome Preset Selector")

	// Загружаем пресеты из GitHub и конвертируем их в TXT
	err = modules.FetchAndConvertBiomeBrushes()
	if err != nil {
		log.Fatalf("Ошибка при загрузке пресетов с GitHub: %v", err)
	}

	// Загружаем список доступных пресетов из локальной папки
	presets, err := modules.FetchAvailablePresets()
	if err != nil {
		log.Fatalf("Ошибка при получении списка пресетов: %v", err)
	}

	// Создаем выпадающий список с пресетами
	presetSelect := widget.NewSelect(presets, func(selected string) {
		fmt.Println("Выбран пресет:", selected)
	})

	// Кнопка для применения выбранного пресета
	applyButton := widget.NewButton("Применить пресет", func() {
		selectedPreset := presetSelect.Selected
		if selectedPreset != "" {
			// Формируем путь к выбранному пресету
			txtFilePath := fmt.Sprintf("presets/%s.txt", selectedPreset)

			// Выполняем замену блоков в r4LavaEditor2.sessions.ini на основе выбранного пресета
			modules.ReplaceBlocksInIni(fullPath, txtFilePath, projectName)
		} else {
			fmt.Println("Пожалуйста, выберите пресет для применения.")
		}
	})

	// Создаем интерфейс с выбором пресета и кнопкой для его применения
	content := container.NewVBox(
		widget.NewLabel("Выберите пресет для применения:"),
		presetSelect,
		applyButton,
	)

	// Настраиваем окно и запускаем интерфейс
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 200))
	myWindow.ShowAndRun()
}
