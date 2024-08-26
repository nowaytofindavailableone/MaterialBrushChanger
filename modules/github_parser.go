package modules

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// GitHub API для получения содержимого репозитория
const (
	githubRepoAPI = "https://api.github.com/repos/nowaytofindavailableone/redkit3biometool/contents/biomebrushes"
	presetsFolder = "./presets" // Папка для хранения загруженных пресетов
)

// FetchAndConvertBiomeBrushes загружает JSON файлы с GitHub, конвертирует их в TXT и сохраняет локально
func FetchAndConvertBiomeBrushes() error {
	// Создаем папку presets, если она не существует
	if _, err := os.Stat(presetsFolder); os.IsNotExist(err) {
		err := os.Mkdir(presetsFolder, 0755)
		if err != nil {
			return fmt.Errorf("ошибка при создании директории пресетов: %v", err)
		}
		fmt.Println("Папка 'presets' успешно создана.")
	}

	// Отправляем запрос к API GitHub для получения содержимого папки biomebrushes
	resp, err := http.Get(githubRepoAPI)
	if err != nil {
		return fmt.Errorf("error fetching biome brushes from GitHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: failed to fetch biome brushes from GitHub, status: %s", resp.Status)
	}

	// Читаем ответ и распаковываем JSON
	var contents []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}

	err = json.NewDecoder(resp.Body).Decode(&contents)
	if err != nil {
		return fmt.Errorf("error decoding GitHub API response: %v", err)
	}

	// Обрабатываем JSON файлы
	for _, file := range contents {
		// Проверяем тип файла и расширение, чтобы обработать только JSON файлы
		if filepath.Ext(file.Name) == ".json" && file.Type == "file" {
			// Загружаем содержимое JSON файла
			brush, err := fetchJSONFromURL(file.DownloadURL)
			if err != nil {
				fmt.Printf("error fetching JSON file %s: %v\n", file.Name, err)
				continue
			}

			// Преобразуем JSON в текстовый формат и сохраняем
			txtFileName := strings.Replace(filepath.Base(file.Name), ".json", ".txt", 1)
			err = ConvertJSONToTxt(brush, filepath.Join(presetsFolder, txtFileName))
			if err != nil {
				return fmt.Errorf("error converting JSON to TXT for file %s: %v", file.Name, err)
			}
		}
	}

	return nil
}

func replaceDoubleBrackets(text string) string {
	text = strings.Replace(text, "[[", "[", -1)
	text = strings.Replace(text, "]]", "]", -1)
	return text
}

// fetchJSONFromURL загружает JSON содержимое по URL
func fetchJSONFromURL(url string) (map[string]map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении JSON по URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка: не удалось получить JSON по URL, статус: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении тела ответа: %v", err)
	}

	var data map[string]map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования JSON: %v", err)
	}

	return data, nil
}

// ConvertJSONToTxt преобразует JSON структуру в нужный текстовый формат и сохраняет в файл
func ConvertJSONToTxt(jsonData map[string]map[string]interface{}, outputFileName string) error {
	var sb strings.Builder

	// Определяем фиксированный порядок ключей для правильного вывода
	keysOrder := []string{
		"VerticalMask",
		"VerticalUVMult",
		"VerticalUVScaleMask",
		"HeightLowLimit",
		"Probability",
		"PresetEnabled",
		"HighLimitMask",
		"HorizontalMask",
		"SelectedHorizontalTexture",
		"SlopeThresholdMask",
		"SelectedVerticalTexture",
		"LowLimitMask",
		"SlopeThresholdAction",
		"SlopeThresholdIndex",
		"HeightHighLimit",
	}

	// Преобразуем JSON структуру в нужный текстовый формат
	for i := 1; i <= len(jsonData); i++ { // Гарантируем порядок обработки
		blockKey := fmt.Sprintf("%d", i)
		block := jsonData[blockKey]

		// Печатаем заголовок блока
		if path, exists := block["path"]; exists {
			header := fmt.Sprintf("[%s]\n", replaceDoubleBrackets(fmt.Sprintf("%v", path)))
			sb.WriteString(header)
		}

		// Печатаем параметры в правильном порядке
		for _, key := range keysOrder {
			if val, exists := block[key]; exists {
				// Преобразуем значение в строку и заменяем двойные квадратные скобки
				sb.WriteString(fmt.Sprintf("%s=%v\n", key, val))
			}
		}
		// Удаляем добавление пустой строки
	}

	// Заменяем двойные скобки во всем текстовом содержимом
	finalOutput := replaceDoubleBrackets(sb.String())

	// Сохраняем результат в файл
	err := ioutil.WriteFile(outputFileName, []byte(finalOutput), 0644)
	if err != nil {
		return fmt.Errorf("ошибка записи в файл: %v", err)
	}

	fmt.Printf("Успешно сохранено в %s\n", outputFileName)
	return nil
}

// FetchAvailablePresets возвращает список доступных пресетов из локальной папки
func FetchAvailablePresets() ([]string, error) {
	// Проверяем, существует ли папка
	if _, err := os.Stat(presetsFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("папка 'presets' не найдена")
	}

	// Читаем содержимое папки presets
	files, err := ioutil.ReadDir(presetsFolder)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении директории пресетов: %v", err)
	}

	var presets []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".txt") {
			presetName := strings.TrimSuffix(file.Name(), ".txt")
			presets = append(presets, presetName)
		}
	}

	return presets, nil
}
